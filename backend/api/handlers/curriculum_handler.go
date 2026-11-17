package handlers

import (
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/learnthor/backend/models"
	"github.com/learnthor/backend/services"
)

// CurriculumHandler serves the section/module tree with per-user progress.
type CurriculumHandler struct {
	db *gorm.DB
}

// NewCurriculumHandler constructs a CurriculumHandler.
func NewCurriculumHandler(db *gorm.DB) *CurriculumHandler {
	return &CurriculumHandler{db: db}
}

type moduleInfo struct {
	ID            uuid.UUID      `json:"id"`
	Title         string         `json:"title"`
	SortOrder     int            `json:"sort_order"`
	VocabCount    int            `json:"vocab_count"`
	Locked        bool           `json:"locked"`
	HighScore     int            `json:"high_score"`
	Passed        bool           `json:"passed"`
	PassThreshold int            `json:"pass_threshold"`
	SkillScores   map[string]int `json:"skill_scores"` // per-skill best scores
}

type sectionInfo struct {
	ID            uuid.UUID    `json:"id"`
	Title         string       `json:"title"`
	SortOrder     int          `json:"sort_order"`
	Locked        bool         `json:"locked"`
	IsPremium     bool         `json:"is_premium"`
	MasteryPct    int          `json:"mastery_pct"`
	ModulesTotal  int          `json:"modules_total"`
	ModulesPassed int          `json:"modules_passed"`
	Modules       []moduleInfo `json:"modules"`
}

// GetSections returns all sections with their modules, including each module's
// mastery data and lock status for the authenticated user.
func (h *CurriculumHandler) GetSections(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	// Privileged users (admin / editor) see every module unlocked so they can
	// preview and test all curriculum content from the homepage.
	privileged := services.CanBypassProgression(h.db, userID)

	// Free-tier users are locked out of premium sections (N4/N3/Daily) with an
	// upgrade prompt; premium users and privileged roles see them open.
	var user models.User
	if err := h.db.Select("subscription_tier").First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "an unexpected error occurred",
		})
		return
	}
	isPremiumUser := privileged || user.SubscriptionTier == models.SubscriptionPremium

	// 1. Load all sections with their modules.
	var sections []models.Section
	if err := h.db.
		Where("is_hidden = ?", false).
		Preload("Modules", "is_hidden = ?", false).
		Order("sort_order asc").
		Find(&sections).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "an unexpected error occurred",
		})
		return
	}

	// Sort modules within each section by sort_order.
	for i := range sections {
		sort.Slice(sections[i].Modules, func(a, b int) bool {
			return sections[i].Modules[a].SortOrder < sections[i].Modules[b].SortOrder
		})
	}

	// 2. Collect all module IDs for batch queries.
	var allModuleIDs []uuid.UUID
	for _, sec := range sections {
		for _, mod := range sec.Modules {
			allModuleIDs = append(allModuleIDs, mod.ID)
		}
	}

	// 3. Vocab count per module.
	type modVocabCount struct {
		ModuleID uuid.UUID
		Count    int
	}
	var vocabCounts []modVocabCount
	if len(allModuleIDs) > 0 {
		h.db.Model(&models.VocabItem{}).
			Select("module_id, count(*) as count").
			Where("module_id IN ?", allModuleIDs).
			Group("module_id").
			Scan(&vocabCounts)
	}
	vocabCountMap := make(map[uuid.UUID]int, len(vocabCounts))
	for _, vc := range vocabCounts {
		vocabCountMap[vc.ModuleID] = vc.Count
	}

	// 4. User's best scores for all modules.
	var progList []models.ModuleProgress
	if len(allModuleIDs) > 0 {
		h.db.Where("user_id = ? AND module_id IN ?", userID, allModuleIDs).
			Find(&progList)
	}
	progMap := make(map[uuid.UUID]models.ModuleProgress, len(progList))
	for _, p := range progList {
		progMap[p.ModuleID] = p
	}

	// 4b. Per-skill progress.
	var skillProgList []models.ModuleSkillProgress
	if len(allModuleIDs) > 0 {
		h.db.Where("user_id = ? AND module_id IN ?", userID, allModuleIDs).
			Find(&skillProgList)
	}
	// skillScoreMap[moduleID][skillType] = highScore
	skillScoreMap := make(map[uuid.UUID]map[string]int)
	for _, sp := range skillProgList {
		if skillScoreMap[sp.ModuleID] == nil {
			skillScoreMap[sp.ModuleID] = map[string]int{}
		}
		skillScoreMap[sp.ModuleID][sp.SkillType] = sp.HighScore
	}

	// 5. Build response applying lock logic.
	//    A section is locked when its predecessor has any unfinished module.
	//    A module is locked when its predecessor in the same section is unfinished.
	result := make([]sectionInfo, 0, len(sections))
	for si, sec := range sections {
		// Premium sections lock free-tier users regardless of progression.
		premiumLocked := sec.IsPremium && !isPremiumUser

		sectionLocked := premiumLocked
		if si > 0 && !privileged && !premiumLocked {
			for _, pm := range sections[si-1].Modules {
				if progMap[pm.ID].HighScore < pm.PassThreshold {
					sectionLocked = true
					break
				}
			}
		}

		modInfos := make([]moduleInfo, 0, len(sec.Modules))
		modulesPassed := 0
		for mi, mod := range sec.Modules {
			modLocked := sectionLocked
			if !modLocked && mi > 0 && !privileged {
				prevMod := sec.Modules[mi-1]
				if progMap[prevMod.ID].HighScore < prevMod.PassThreshold {
					modLocked = true
				}
			}
			prog := progMap[mod.ID]
			passed := prog.HighScore >= mod.PassThreshold
			if passed {
				modulesPassed++
			}
			skillScores := skillScoreMap[mod.ID]
			if skillScores == nil {
				skillScores = map[string]int{}
			}
			modInfos = append(modInfos, moduleInfo{
				ID:            mod.ID,
				Title:         mod.Title,
				SortOrder:     mod.SortOrder,
				VocabCount:    vocabCountMap[mod.ID],
				Locked:        modLocked,
				HighScore:     prog.HighScore,
				Passed:        passed,
				PassThreshold: mod.PassThreshold,
				SkillScores:   skillScores,
			})
		}

		masteryPct := 0
		if len(sec.Modules) > 0 {
			masteryPct = (modulesPassed * 100) / len(sec.Modules)
		}

		result = append(result, sectionInfo{
			ID:            sec.ID,
			Title:         sec.Title,
			SortOrder:     sec.SortOrder,
			Locked:        sectionLocked,
			IsPremium:     sec.IsPremium,
			MasteryPct:    masteryPct,
			ModulesTotal:  len(sec.Modules),
			ModulesPassed: modulesPassed,
			Modules:       modInfos,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"count":    len(result),
		"sections": result,
	})
}
