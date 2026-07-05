package postgres

import (
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("catalog image admin transitions", func() {
	It("should require an uploaded reviewable candidate before making an image primary", func() {
		key := "catalog-meals/a.jpg"
		mime, sha, license, attribution := "image/jpeg", "abc", "CC BY", "Photographer / CC BY"
		width, height := 800, 600
		valid := model.PrebuiltMealImage{MatchStatus: "auto_accepted", MinioObjectKey: &key, MIMEType: &mime, Width: &width, Height: &height, SHA256: &sha, LicenseName: &license, AttributionText: &attribution}
		review := valid
		review.MatchStatus = "needs_review"

		tests := []struct {
			name  string
			image model.PrebuiltMealImage
			valid bool
		}{
			{name: "accepted upload", image: valid, valid: true},
			{name: "review upload", image: review, valid: true},
			{name: "missing upload", image: model.PrebuiltMealImage{MatchStatus: "needs_review"}, valid: false},
			{name: "rejected", image: model.PrebuiltMealImage{MatchStatus: "rejected", MinioObjectKey: &key}, valid: false},
		}

		for _, test := range tests {
			err := validateImageAction(test.image, interfaces.CatalogImageAction{Action: "approve"})
			if test.valid {
				Expect(err).NotTo(HaveOccurred(), test.name)
			} else {
				Expect(err).To(HaveOccurred(), test.name)
			}
		}
	})
})
