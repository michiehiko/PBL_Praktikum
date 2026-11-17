package service

import (
	"fmt"

	"latihan-fiber/app/model"
)

// BuildQuotaMessage menyusun email untuk event course.quota_changed.
// fungsi ini tidak menyentuh database maupun email.
func BuildQuotaMessage(e model.QuotaChangedEvent, user model.User) model.Message {
	course := e.CourseCode + " " + e.CourseName
	return model.Message{
		To:      user.Email,
		Subject: fmt.Sprintf("KRS: kuota %s berubah menjadi %d", course, e.Quota),
		Body:    fmt.Sprintf("Halo %s, kuota kelas %s kini menjadi %d.", user.Username, course, e.Quota),
	}
}