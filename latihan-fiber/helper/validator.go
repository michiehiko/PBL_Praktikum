package helper

import (
	"errors"
	"reflect"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New()

	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return field.Name
		}
		return name
	})

	_ = v.RegisterValidation("nospace", func(fl validator.FieldLevel) bool {
		return !strings.ContainsAny(fl.Field().String(), " \t\n\r")
	})

	_ = v.RegisterValidation("nim", func(fl validator.FieldLevel) bool {
		val := fl.Field().String()
		for _, r := range val {
			if !unicode.IsDigit(r) {
				return false
			}
		}
		return true
	})

	return v
}

func ValidateStruct(s any) map[string]string {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	var invalid *validator.InvalidValidationError
	if errors.As(err, &invalid) {
		return map[string]string{"_": "objek yang divalidasi tidak sah"}
	}

	var fieldErrors validator.ValidationErrors
	if !errors.As(err, &fieldErrors) {
		return map[string]string{"_": "validasi gagal"}
	}

	result := make(map[string]string, len(fieldErrors))
	for _, fe := range fieldErrors {
		if _, exists := result[fe.Field()]; !exists {
			result[fe.Field()] = messageFor(fe)
		}
	}
	return result
}

func messageFor(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "wajib diisi"
	case "min":
		switch fe.Kind() {
		case reflect.String:
			return "minimal " + fe.Param() + " karakter"
		case reflect.Slice, reflect.Array, reflect.Map:
			return "minimal " + fe.Param() + " item"
		}
		return "nilai minimal " + fe.Param()
	case "max":
		switch fe.Kind() {
		case reflect.String:
			return "maksimal " + fe.Param() + " karakter"
		case reflect.Slice, reflect.Array, reflect.Map:
			return "maksimal " + fe.Param() + " item"
		}
		return "nilai maksimal " + fe.Param()
	case "alphanum":
		return "hanya boleh berisi huruf dan angka"
	case "nospace":
		return "tidak boleh mengandung spasi"
	case "oneof":
		return "harus salah satu dari: " + strings.ReplaceAll(fe.Param(), " ", ", ")
	case "nim":
		return "NIM hanya boleh berisi angka"
	default:
		return "tidak memenuhi aturan " + fe.Tag()
	}
}