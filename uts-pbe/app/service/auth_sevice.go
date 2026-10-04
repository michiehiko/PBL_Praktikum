package service

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"uts-pbe/app/model"
	"uts-pbe/app/repository"
	"uts-pbe/helper"
)

const refreshTokenBytes = 32

type AuthService struct {
	users      repository.UserRepository
	tokens     repository.TokenRepository
	students   repository.StudentRepository
	jwt        *helper.JWTManager
	refreshTTL time.Duration
}

func NewAuthService(
	users repository.UserRepository,
	tokens repository.TokenRepository,
	students repository.StudentRepository,
	jwtManager *helper.JWTManager,
	refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		users: users, tokens: tokens, students: students, jwt: jwtManager, refreshTTL: refreshTTL,
	}
}

func (s *AuthService) Login(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	user, err := s.users.FindByEmail(ctx, strings.TrimSpace(req.Email))
	if err != nil {
		helper.VerifyDummyPassword(req.Password) // mencegah user enumeration
		return helper.Unauthorized("email atau password salah")
	}

	if !helper.VerifyPassword(user.Password, req.Password) {
		return helper.Unauthorized("email atau password salah")
	}

	pair, err := s.issueTokenPair(ctx, user)
	if err != nil {
		return helper.Internal(err)
	}

	// Response khusus sesuai tugas uts
	return helper.Success(c, fiber.StatusOK, "login berhasil", fiber.Map{
		"access_token": pair.AccessToken,
		"token_type":   pair.TokenType,
		"expires_in":   pair.ExpiresIn,
		"user": fiber.Map{
			"id":    user.ID,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

func (s *AuthService) Refresh(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if strings.TrimSpace(req.RefreshToken) == "" {
		return helper.BadRequest("refresh_token wajib diisi")
	}

	hash := helper.SHA256Hex(req.RefreshToken)
	stored, err := s.tokens.FindActive(ctx, hash)
	if err != nil {
		return helper.Unauthorized("refresh token tidak valid atau sudah kedaluwarsa")
	}

	user, err := s.users.FindByID(ctx, stored.UserID)
	if err != nil {
		return helper.Unauthorized("akun tidak dapat dipakai")
	}

	_ = s.tokens.Revoke(ctx, hash)

	pair, err := s.issueTokenPair(ctx, user)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "token berhasil diperbarui", pair)
}

func (s *AuthService) Logout(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if strings.TrimSpace(req.RefreshToken) != "" {
		_ = s.tokens.Revoke(ctx, helper.SHA256Hex(req.RefreshToken))
	}
	return helper.Success(c, fiber.StatusOK, "logout berhasil", nil)
}

func (s *AuthService) Me(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	authUser, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	user, err := s.users.FindByID(ctx, authUser.UserID)
	if err != nil {
		return helper.Unauthorized("user tidak ditemukan")
	}

	// Response khusus sesuai tugas UTS (Sisipkan data mahasiswa)
	response := fiber.Map{
		"id":    user.ID,
		"email": user.Email,
		"role":  user.Role,
	}

	if user.Role == "mahasiswa" {
		student, err := s.students.FindByUserID(ctx, user.ID)
		if err == nil {
			response["student"] = student
		}
	}

	return helper.Success(c, fiber.StatusOK, "profil berhasil diambil", response)
}

func (s *AuthService) issueTokenPair(ctx context.Context, user model.User) (model.TokenPair, error) {
	accessToken, err := s.jwt.GenerateAccess(user)
	if err != nil {
		return model.TokenPair{}, err
	}
	refreshToken, err := helper.RandomToken(refreshTokenBytes)
	if err != nil {
		return model.TokenPair{}, err
	}
	err = s.tokens.Save(ctx, model.RefreshToken{
		UserID:    user.ID,
		TokenHash: helper.SHA256Hex(refreshToken),
		ExpiresAt: time.Now().Add(s.refreshTTL),
	})
	if err != nil {
		return model.TokenPair{}, err
	}
	return model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.jwt.AccessTTL().Seconds()),
	}, nil
}
