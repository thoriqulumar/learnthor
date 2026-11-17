package services

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/learnthor/backend/models"
)

// addModule inserts an extra module into the same section as fx.Module.
func addModule(t *testing.T, db *gorm.DB, fx curriculumFixture, title string, sortOrder int) models.Module {
	t.Helper()
	m := models.Module{
		SectionID:     fx.Section.ID,
		Title:         title,
		SortOrder:     sortOrder,
		PassThreshold: 90,
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("add module: %v", err)
	}
	return m
}

// setProgress upserts a ModuleProgress row with the given high score.
func setProgress(t *testing.T, db *gorm.DB, userID, moduleID uuid.UUID, highScore int) {
	t.Helper()
	p := models.ModuleProgress{UserID: userID, ModuleID: moduleID, HighScore: highScore}
	if err := db.Save(&p).Error; err != nil {
		t.Fatalf("set progress: %v", err)
	}
}

func TestCheckModuleAccess_FirstModuleAlwaysAllowed(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 1)
	user := createTestUser(t, db, models.UserRoleStudent)

	svc := NewGatingService(db)
	res, err := svc.CheckModuleAccess(user.ID, fx.Module.ID)
	if err != nil {
		t.Fatalf("CheckModuleAccess: %v", err)
	}
	if !res.Allowed {
		t.Fatalf("first module should be allowed, got denied: %s", res.Reason)
	}
}

func TestCheckModuleAccess_DeniedWithoutPredecessorProgress(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 1)
	second := addModule(t, db, fx, "Module 2", 2)
	user := createTestUser(t, db, models.UserRoleStudent)

	svc := NewGatingService(db)
	res, err := svc.CheckModuleAccess(user.ID, second.ID)
	if err != nil {
		t.Fatalf("CheckModuleAccess: %v", err)
	}
	if res.Allowed {
		t.Fatal("second module should be denied without predecessor progress")
	}
	if res.RequiredModuleID != fx.Module.ID {
		t.Fatalf("required module = %s, want predecessor %s", res.RequiredModuleID, fx.Module.ID)
	}
	if res.Reason == "" {
		t.Fatal("denied result should carry a human-readable reason")
	}
}

func TestCheckModuleAccess_DeniedBelowThreshold(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 1)
	second := addModule(t, db, fx, "Module 2", 2)
	user := createTestUser(t, db, models.UserRoleStudent)
	setProgress(t, db, user.ID, fx.Module.ID, 50) // below 90 threshold

	svc := NewGatingService(db)
	res, err := svc.CheckModuleAccess(user.ID, second.ID)
	if err != nil {
		t.Fatalf("CheckModuleAccess: %v", err)
	}
	if res.Allowed {
		t.Fatal("should be denied when predecessor score is below threshold")
	}
	if res.UserHighScore != 50 {
		t.Fatalf("UserHighScore = %d, want 50", res.UserHighScore)
	}
}

func TestCheckModuleAccess_AllowedAtThreshold(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 1)
	second := addModule(t, db, fx, "Module 2", 2)
	user := createTestUser(t, db, models.UserRoleStudent)
	setProgress(t, db, user.ID, fx.Module.ID, 90) // exactly at threshold

	svc := NewGatingService(db)
	res, err := svc.CheckModuleAccess(user.ID, second.ID)
	if err != nil {
		t.Fatalf("CheckModuleAccess: %v", err)
	}
	if !res.Allowed {
		t.Fatalf("should be allowed at threshold, denied: %s", res.Reason)
	}
}

func TestCheckModuleAccess_PrivilegedBypass(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 1)
	second := addModule(t, db, fx, "Module 2", 2)

	svc := NewGatingService(db)
	for _, role := range []models.UserRole{models.UserRoleAdmin, models.UserRoleEditor} {
		u := createTestUser(t, db, role)
		res, err := svc.CheckModuleAccess(u.ID, second.ID)
		if err != nil {
			t.Fatalf("%s: %v", role, err)
		}
		if !res.Allowed {
			t.Fatalf("%s should bypass gating, denied: %s", role, res.Reason)
		}
	}
}

func TestCheckModuleAccess_PrivilegedStillGetsNotFound(t *testing.T) {
	db := newTestDB(t)
	admin := createTestUser(t, db, models.UserRoleAdmin)

	svc := NewGatingService(db)
	_, err := svc.CheckModuleAccess(admin.ID, uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound even for admin", err)
	}
}

func TestCheckModuleAccess_NotFound(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, models.UserRoleStudent)

	svc := NewGatingService(db)
	_, err := svc.CheckModuleAccess(user.ID, uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestCheckModuleAccess_FreeUserDeniedOnPremiumSection(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 1)
	// Mark the section premium.
	db.Model(&fx.Section).Update("is_premium", true)

	user := createTestUser(t, db, models.UserRoleStudent)
	svc := NewGatingService(db)

	res, err := svc.CheckModuleAccess(user.ID, fx.Module.ID)
	if err != nil {
		t.Fatalf("CheckModuleAccess: %v", err)
	}
	if res.Allowed {
		t.Fatal("free user should be denied on a premium section")
	}
	if !res.RequiresPremium {
		t.Fatal("denial should be flagged RequiresPremium, not progression-locked")
	}
}

func TestCheckModuleAccess_PremiumUserAllowedOnPremiumSection(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 1)
	db.Model(&fx.Section).Update("is_premium", true)

	user := createTestUser(t, db, models.UserRoleStudent)
	db.Model(&user).Update("subscription_tier", models.SubscriptionPremium)

	svc := NewGatingService(db)
	res, err := svc.CheckModuleAccess(user.ID, fx.Module.ID)
	if err != nil {
		t.Fatalf("CheckModuleAccess: %v", err)
	}
	if !res.Allowed {
		t.Fatalf("premium user should be allowed, denied: %s", res.Reason)
	}
}

func TestCheckModuleAccess_AdminBypassesPremiumGate(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 1)
	db.Model(&fx.Section).Update("is_premium", true)

	admin := createTestUser(t, db, models.UserRoleAdmin)
	svc := NewGatingService(db)

	res, err := svc.CheckModuleAccess(admin.ID, fx.Module.ID)
	if err != nil {
		t.Fatalf("CheckModuleAccess: %v", err)
	}
	if !res.Allowed {
		t.Fatalf("admin should bypass premium gate, denied: %s", res.Reason)
	}
}

func TestIsPrivilegedUser(t *testing.T) {
	db := newTestDB(t)
	admin := createTestUser(t, db, models.UserRoleAdmin)
	editor := createTestUser(t, db, models.UserRoleEditor)
	student := createTestUser(t, db, models.UserRoleStudent)

	if !IsPrivilegedUser(db, admin.ID) {
		t.Fatal("admin should be privileged")
	}
	if !IsPrivilegedUser(db, editor.ID) {
		t.Fatal("editor should be privileged")
	}
	if IsPrivilegedUser(db, student.ID) {
		t.Fatal("student must not be privileged")
	}
	if IsPrivilegedUser(db, uuid.New()) {
		t.Fatal("nonexistent user must not be privileged")
	}
}

func TestHiddenCurriculumIsNotAccessible(t *testing.T) {
	database := newTestDB(t)
	fx := seedCurriculum(t, database, 3)
	admin := createTestUser(t, database, models.UserRoleAdmin)
	if err := database.Model(&fx.Module).Update("is_hidden", true).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewGatingService(database).CheckModuleAccess(admin.ID, fx.Module.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("hidden module error = %v, want ErrNotFound", err)
	}
	if err := database.Model(&fx.Section).Update("is_hidden", true).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewGatingService(database).CheckSectionAccess(admin.ID, fx.Section.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("hidden section error = %v, want ErrNotFound", err)
	}
}
