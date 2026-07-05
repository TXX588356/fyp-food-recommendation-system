package preference

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPreferenceService(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Preference Service Suite")
}

var _ = Describe("meal preference validation", func() {
	It("shoulf accept current frontend category codes", func() {
		tags := []string{
			"singaporean",
			"rice_dishes",
			"noodle_dishes",
			"condiments_sauces",
			"poultry",
			"tofu_soy",
			"nuts_seeds",
		}

		Expect(validateMealPreferences(tags)).To(Succeed())
	})

	It("should check dietary restrictions using current category codes", func() {
		err := validateMealPreferencesAgainstDietaryRestrictions([]string{"nuts_seeds"}, []string{"nut_free"})

		Expect(err).To(HaveOccurred())
	})
})
