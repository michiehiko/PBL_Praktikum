package service

import (
	"latihan-fiber/app/model"
	"latihan-fiber/helper"
)
// memutuskan apakah user saat ini boleh mengakses data student tertentu
func CanAccessStudent(
	current model.AuthUser,
	ownerID int,
	perms *helper.PermissionSet,
	anyPermission string,
) bool {
	if current.UserID == ownerID {
		return true 
	}
	// kalau bukan owner, periksa apakah user memiliki permission untuk mengakses data yang lain
	return perms.Can(current.Role, anyPermission)
}

