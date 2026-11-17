package services

import (
	"errors"
	"testing"

	"github.com/learnthor/backend/models"
)

func TestPersonalPremiumIsStudentOnlyAndExclusive(t *testing.T) {
	database := newTestDB(t)
	owner := createTestUser(t, database, models.UserRoleAdmin)
	other := createTestUser(t, database, models.UserRoleStudent)
	if err := database.Model(&other).Updates(map[string]any{"subscription_tier": models.SubscriptionPremium, "progression_bypass": true}).Error; err != nil {
		t.Fatal(err)
	}
	granted, err := GrantPersonalPremium(database, owner.Username)
	if err != nil {
		t.Fatal(err)
	}
	if granted.Role != models.UserRoleStudent || granted.SubscriptionTier != models.SubscriptionPremium || !granted.ProgressionBypass {
		t.Fatalf("wrong access: %+v", granted)
	}
	if IsPrivilegedUser(database, owner.ID) {
		t.Fatal("personal account gained admin privileges")
	}
	if !CanBypassProgression(database, owner.ID) {
		t.Fatal("personal account still locked")
	}
	if CanBypassProgression(database, other.ID) {
		t.Fatal("exception leaked to other account")
	}
	if err := database.First(&other, "id = ?", other.ID).Error; err != nil {
		t.Fatal(err)
	}
	if other.SubscriptionTier != models.SubscriptionPremium {
		t.Fatal("other user's premium removed")
	}
	if _, err := GrantPersonalPremium(database, "missing-user"); err == nil {
		t.Fatal("missing user accepted")
	}
	if !CanBypassProgression(database, owner.ID) {
		t.Fatal("failed grant revoked existing exception")
	}
}

func TestPremiumStudentsMustFinishPreviousSections(t *testing.T) {
	database := newTestDB(t)
	fx := seedCurriculum(t, database, 4)
	owner := createTestUser(t, database, models.UserRoleStudent)
	regular := createTestUser(t, database, models.UserRoleStudent)
	if _, err := GrantPersonalPremium(database, owner.Username); err != nil {
		t.Fatal(err)
	}
	if err := database.Model(&regular).Update("subscription_tier", models.SubscriptionPremium).Error; err != nil {
		t.Fatal(err)
	}
	section := models.Section{CourseID: fx.Course.ID, Title: "Next section", SortOrder: fx.Section.SortOrder + 1, IsPremium: true}
	if err := database.Create(&section).Error; err != nil {
		t.Fatal(err)
	}
	module := models.Module{SectionID: section.ID, Title: "Next module", SortOrder: 1, PassThreshold: 90}
	if err := database.Create(&module).Error; err != nil {
		t.Fatal(err)
	}
	gate := NewGatingService(database)
	result, err := gate.CheckModuleAccess(regular.ID, module.ID)
	if err != nil || result.Allowed || result.RequiresPremium {
		t.Fatalf("premium must retain progression lock: %+v %v", result, err)
	}
	result, err = gate.CheckModuleAccess(owner.ID, module.ID)
	if err != nil || !result.Allowed {
		t.Fatalf("personal exception denied: %+v %v", result, err)
	}
	if _, err := NewSessionService(database).StartSession(StartSessionRequest{UserID: regular.ID, ModuleID: module.ID}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("session bypassed gate: %v", err)
	}
	if _, err := NewQuizService(database).GenerateSectionQuiz(GenerateSectionQuizRequest{UserID: regular.ID, SectionID: section.ID}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("section quiz bypassed gate: %v", err)
	}
	setProgress(t, database, regular.ID, fx.Module.ID, 90)
	result, err = gate.CheckModuleAccess(regular.ID, module.ID)
	if err != nil || !result.Allowed {
		t.Fatalf("completed prerequisite still locked: %+v %v", result, err)
	}
}

func TestPersonalExceptionRequiresPremium(t *testing.T) {
	database := newTestDB(t)
	user := createTestUser(t, database, models.UserRoleStudent)
	if err := database.Model(&user).Update("progression_bypass", true).Error; err != nil {
		t.Fatal(err)
	}
	if CanBypassProgression(database, user.ID) {
		t.Fatal("free student received premium bypass")
	}
}
