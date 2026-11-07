package sloc

// Language_For_Test gives black-box specification tests direct access to the static
// seed selector without adding fixed-value production boundaries.
func Language_For_Test(name Known_Name) (language Seeded_Language) {
	return language_seed(name)
}

// Classify_File drives the production byte classifier through its real boundary.
func Classify_File(input Classify_File_Input) (counts File_Partition) {
	return classify(File_Classifier{Kind: FILE_CLASSIFIER_KIND_BYTES}, input)
}

// Language_For_Filename exposes the filename matcher without a second seed boundary.
func Language_For_Filename(
	name File_Name,
) (language Language, recognized Recognition) {
	seed_name := Known_Name("")
	recognized = language_name_for_filename(name, func(match Known_Name) {
		seed_name = match
	})
	if !recognized {
		return Language{}, false
	}
	return Language(language_seed(seed_name)), true
}
