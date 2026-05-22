package recipe

import (
	"fmt"

	"github.com/yangyifan18/dotvibe/backup"
)

type recipeManifestError struct {
	code    string
	message string
}

func (e recipeManifestError) Error() string {
	return e.message
}

func validateRecipeManifest(manifest *backup.Manifest) error {
	if manifest == nil {
		return recipeManifestError{code: "invalid_manifest", message: "archive missing manifest"}
	}
	if manifest.ArchiveKind != backup.ArchiveKindRecipe {
		return recipeManifestError{code: "not_recipe", message: "archive is not a dotvibe recipe"}
	}
	if manifest.Recipe == nil {
		return recipeManifestError{code: "missing_recipe_metadata", message: "recipe archive missing manifest.recipe metadata"}
	}
	if manifest.Recipe.Schema != backup.RecipeSchemaV1 {
		return recipeManifestError{
			code:    "schema_mismatch",
			message: fmt.Sprintf("unsupported recipe schema %q", manifest.Recipe.Schema),
		}
	}
	return nil
}
