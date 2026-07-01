package postgres

import (
	"testing"

	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"
)

func TestImageCanBecomePrimaryRequiresUploadedReviewableCandidate(t *testing.T) {
	key := "catalog-meals/a.jpg"
	mime, sha, license, attribution := "image/jpeg", "abc", "CC BY", "Photographer / CC BY"
	width, height := 800, 600
	valid := model.PrebuiltMealImage{MatchStatus: "auto_accepted", MinioObjectKey: &key, MIMEType: &mime, Width: &width, Height: &height, SHA256: &sha, LicenseName: &license, AttributionText: &attribution}
	review := valid
	review.MatchStatus = "needs_review"
	for _, test := range []struct {
		name  string
		image model.PrebuiltMealImage
		valid bool
	}{
		{name: "accepted upload", image: valid, valid: true},
		{name: "review upload", image: review, valid: true},
		{name: "missing upload", image: model.PrebuiltMealImage{MatchStatus: "needs_review"}, valid: false},
		{name: "rejected", image: model.PrebuiltMealImage{MatchStatus: "rejected", MinioObjectKey: &key}, valid: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateImageAction(test.image, interfaces.CatalogImageAction{Action: "approve"})
			if test.valid && err != nil {
				t.Fatal(err)
			}
			if !test.valid && err == nil {
				t.Fatal("expected transition to fail")
			}
		})
	}
}
