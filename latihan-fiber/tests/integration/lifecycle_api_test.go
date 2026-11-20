//go:build integration

package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"latihan-fiber/app/model"
)

// request mengirim satu request HTTP ke aplikasi uji atas nama user, lalu
// mengembalikan status dan body JSON (body tidak dibaca untuk 204).
func (a api) request(t *testing.T, user model.User, method, path string, payload any) (int, map[string]any) {
	t.Helper()

	token, err := a.jwt.GenerateAccess(model.User{ID: user.ID, Username: user.Username, Role: user.Role})
	if err != nil {
		t.Fatalf("membuat token: %v", err)
	}

	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("menyusun body: %v", err)
		}
		body = bytes.NewReader(raw)
	}

	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.app.Test(req, -1)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return resp.StatusCode, nil
	}

	var decoded map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&decoded)
	return resp.StatusCode, decoded
}

// countOutbox menghitung event tertentu milik seorang user. Bila courseID
// bukan nol, hitungan dibatasi pada kelas itu.
func countOutbox(t *testing.T, userID, courseID int, eventType string) int {
	t.Helper()
	if courseID == 0 {
		return countRows(t,
			`SELECT COUNT(*) FROM outbox
			 WHERE (payload->>'user_id')::INT = $1 AND event_type = $2`,
			userID, eventType)
	}
	return countRows(t,
		`SELECT COUNT(*) FROM outbox
		 WHERE (payload->>'user_id')::INT = $1 AND event_type = $2
		   AND (payload->>'course_id')::INT = $3`,
		userID, eventType, courseID)
}

// TestSwapEndpoint_FullDestinationKeepsOriginal memeriksa janji di anotasi
// Swap: "Bila kelas baru penuh, mahasiswa tetap di kelas lama."
//
// Yang diperiksa bukan hanya status 409, tetapi juga keadaan database
// sesudahnya. Endpoint yang membalas 409 namun sudah terlanjur menghapus
// pendaftaran lama tetap merusak data.
func TestSwapEndpoint_FullDestinationKeepsOriginal(t *testing.T) {
	a := newAPI(t)
	origin := seedCourse(t, 3, 40)
	destination := seedCourse(t, 3, 1) // kuota satu kursi
	student := seedStudent(t)
	other := seedStudent(t)

	if status, _ := a.enroll(t, student, origin.ID); status != http.StatusCreated {
		t.Fatalf("pendaftaran ke kelas asal: status = %d, seharusnya 201", status)
	}
	// Isi satu-satunya kursi di kelas tujuan dengan mahasiswa lain.
	if status, _ := a.enroll(t, other, destination.ID); status != http.StatusCreated {
		t.Fatalf("mengisi kelas tujuan: status = %d, seharusnya 201", status)
	}

	originBefore := countRows(t, `SELECT enrolled_count FROM courses WHERE id = $1`, origin.ID)
	destinationBefore := countRows(t, `SELECT enrolled_count FROM courses WHERE id = $1`, destination.ID)
	swappedBefore := countOutbox(t, student.ID, 0, model.EventEnrollmentSwapped)

	status, body := a.request(t, student, http.MethodPost, "/api/v1/enrollments/swap",
		map[string]int{"from_course_id": origin.ID, "to_course_id": destination.ID})

	if status != http.StatusConflict {
		t.Fatalf("status = %d, seharusnya 409 (body: %v)", status, body)
	}
	if code, _ := body["code"].(string); code != "COURSE_FULL" {
		t.Errorf("code = %q, seharusnya COURSE_FULL", code)
	}

	if n := countRows(t,
		`SELECT COUNT(*) FROM enrollments WHERE course_id = $1 AND user_id = $2`,
		origin.ID, student.ID); n != 1 {
		t.Errorf("baris enrollment di kelas asal = %d, seharusnya 1 (mahasiswa tidak boleh tercabut dari kelas lama)", n)
	}
	if n := countRows(t,
		`SELECT COUNT(*) FROM enrollments WHERE course_id = $1 AND user_id = $2`,
		destination.ID, student.ID); n != 0 {
		t.Errorf("baris enrollment di kelas tujuan = %d, seharusnya 0", n)
	}
	if got := countRows(t, `SELECT enrolled_count FROM courses WHERE id = $1`, origin.ID); got != originBefore {
		t.Errorf("enrolled_count kelas asal = %d, seharusnya tetap %d", got, originBefore)
	}
	if got := countRows(t, `SELECT enrolled_count FROM courses WHERE id = $1`, destination.ID); got != destinationBefore {
		t.Errorf("enrolled_count kelas tujuan = %d, seharusnya tetap %d", got, destinationBefore)
	}
	if got := countOutbox(t, student.ID, 0, model.EventEnrollmentSwapped); got != swappedBefore {
		t.Errorf("event enrollment.swapped = %d, seharusnya tetap %d (pindah yang gagal tidak boleh mencatat event)", got, swappedBefore)
	}
}

// TestDropEndpoint_WritesOneDroppedEvent memeriksa DELETE /courses/:id/enroll:
// 204, baris enrollment hilang, enrolled_count berkurang satu, dan outbox
// berisi TEPAT satu event enrollment.dropped.
func TestDropEndpoint_WritesOneDroppedEvent(t *testing.T) {
	a := newAPI(t)
	course := seedCourse(t, 3, 40)
	student := seedStudent(t)

	if status, _ := a.enroll(t, student, course.ID); status != http.StatusCreated {
		t.Fatalf("pendaftaran: status = %d, seharusnya 201", status)
	}

	countBefore := countRows(t, `SELECT enrolled_count FROM courses WHERE id = $1`, course.ID)
	droppedBefore := countOutbox(t, student.ID, course.ID, model.EventEnrollmentDropped)

	status, _ := a.request(t, student, http.MethodDelete,
		"/api/v1/courses/"+strconv.Itoa(course.ID)+"/enroll", nil)
	if status != http.StatusNoContent {
		t.Fatalf("status = %d, seharusnya 204", status)
	}

	if n := countRows(t,
		`SELECT COUNT(*) FROM enrollments WHERE course_id = $1 AND user_id = $2`,
		course.ID, student.ID); n != 0 {
		t.Errorf("baris enrollment setelah keluar = %d, seharusnya 0", n)
	}
	if got := countRows(t, `SELECT enrolled_count FROM courses WHERE id = $1`, course.ID); got != countBefore-1 {
		t.Errorf("enrolled_count = %d, seharusnya %d", got, countBefore-1)
	}
	if got := countOutbox(t, student.ID, course.ID, model.EventEnrollmentDropped); got != droppedBefore+1 {
		t.Errorf("event enrollment.dropped = %d, seharusnya tepat %d", got, droppedBefore+1)
	}
}
