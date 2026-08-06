package simulation_test

import (
	"fmt"
	"testing"

	snap "local/james-orcales/shared/snap/default"
)

// Test_Main_Reproduces_Each_Stream freezes each named stream before the harness consumes it.
func Test_Main_Reproduces_Each_Stream(t *testing.T) {
	t.Parallel()
	entries := []struct {
		Name     string
		Snapshot snap.Snapshot
	}{
		{Name: "configuration_home", Snapshot: snap.Init(`[4583841352596227090 10276703781306274186 3382288581075887249]`)},
		{Name: "configuration_operating_system", Snapshot: snap.Init(`[16521629639822800165 2736766839171971727 8218131234678978069]`)},
		{Name: "configuration_cargo", Snapshot: snap.Init(`[14827925873436202750 9792458149915043264 11680647098731090832]`)},
		{Name: "configuration_data", Snapshot: snap.Init(`[18167052426772697035 2590586310737351642 17452662854743304688]`)},
		{Name: "configuration_color", Snapshot: snap.Init(`[4948088219520943511 12929637487627772415 2806288048679277221]`)},
		{Name: "configuration_installed", Snapshot: snap.Init(`[4630230380959726927 2183481035415131706 12204932232381385668]`)},
		{Name: "configuration_identifier", Snapshot: snap.Init(`[4736888410879062190 5885079701376307701 11643696404283133570]`)},
		{Name: "configuration_failure", Snapshot: snap.Init(`[3791664082557377893 18379135737203882290 7285808582708187488]`)},
		{Name: "configuration_version", Snapshot: snap.Init(`[7162771157285493794 4199145460316734671 6288700608161180904]`)},
		{Name: "configuration_dotfile_bytes", Snapshot: snap.Init(`[14656580940919077716 1398853343853011766 17788998724064846401]`)},
		{Name: "configuration_probe_output_bytes", Snapshot: snap.Init(`[18069036341845465746 17600139498230406601 10733461417612725260]`)},
		{Name: "configuration_ghostty_verified", Snapshot: snap.Init(`[11355523436760237034 395030742669538522 8282897137189979924]`)},
		{Name: "configuration_empty_version", Snapshot: snap.Init(`[11998781556697223856 18245740488587934280 17897967630694177356]`)},
		{Name: "configuration_path_boundary", Snapshot: snap.Init(`[10364086866226795393 4417789187567558222 5678823332787175929]`)},
		{Name: "configuration_clock_epoch", Snapshot: snap.Init(`[18363995701544801800 10103203485576915068 14516345905584565196]`)},
		{Name: "configuration_io_chunk_bytes", Snapshot: snap.Init(`[13591918114860449320 12453818624701409723 3472577221784257154]`)},
		{Name: "configuration_io_fault", Snapshot: snap.Init(`[15603201849696471663 5702477641072149352 5775392054570315857]`)},
		{Name: "configuration_io_fault_target", Snapshot: snap.Init(`[10833883172859904948 6768503113610058037 3219676999931289914]`)},
		{Name: "configuration_dotfile_content", Snapshot: snap.Init(`[11796264697881919910 12552817528799432530 3513316271812069713]`)},
		{Name: "configuration_probe_output", Snapshot: snap.Init(`[15108500525951294641 15573904313990914929 12687868308533899731]`)},
		{Name: "configuration_io_fault_site", Snapshot: snap.Init(`[17950582978352256815 2026077571228253442 889056340853694291]`)},
		{Name: "configuration_profile_file", Snapshot: snap.Init(`[999780601833595542 14019286928292975892 14216284253079574424]`)},
		{Name: "configuration_configuration_file", Snapshot: snap.Init(`[7392291134903683545 11100547857027873761 13345321761293874464]`)},
		{Name: "configuration_ignored_file", Snapshot: snap.Init(`[13086712658983199744 10094834749596183487 7846495417526405648]`)},
		{Name: "configuration_boundary_file", Snapshot: snap.Init(`[4559081881189776099 1019831100200799914 1869982986640671037]`)},
		{Name: "configuration_boundary_directory", Snapshot: snap.Init(`[5127507593772685229 13234648099120621972 13401723943738923710]`)},
		{Name: "configuration_io_witness", Snapshot: snap.Init(`[8683774887050828851 18428975865289852696 17315395833744861666]`)},
		{Name: "configuration_nested_file", Snapshot: snap.Init(`[5933024012086848208 9388211317721603719 17103193130625041207]`)},
		{Name: "file_profile", Snapshot: snap.Init(`[16521629639822800165 2736766839171971727 8218131234678978069]`)},
		{Name: "file_configuration", Snapshot: snap.Init(`[14827925873436202750 9792458149915043264 11680647098731090832]`)},
		{Name: "file_ignored", Snapshot: snap.Init(`[18167052426772697035 2590586310737351642 17452662854743304688]`)},
		{Name: "file_boundary", Snapshot: snap.Init(`[4948088219520943511 12929637487627772415 2806288048679277221]`)},
		{Name: "file_nested_boundary", Snapshot: snap.Init(`[4630230380959726927 2183481035415131706 12204932232381385668]`)},
		{Name: "file_io_witness", Snapshot: snap.Init(`[4736888410879062190 5885079701376307701 11643696404283133570]`)},
		{Name: "file_font", Snapshot: snap.Init(`[3791664082557377893 18379135737203882290 7285808582708187488]`)},
		{Name: "mutation", Snapshot: snap.Init(`[6409272458699751175 13022637563981439720 2480867791675319432]`)},
	}
	for _, entry := range entries {
		entry := entry
		t.Run(entry.Name, func(t *testing.T) {
			first_values := [HARNESS_STREAM_SNAPSHOT_SEEDS]uint64{}
			for seed := uint64(0); seed < uint64(len(first_values)); seed++ {
				first_values[seed] = harness_stream_first_values(seed)[entry.Name]
			}
			snap.Expect(t, entry.Snapshot, fmt.Sprint(first_values))
		})
	}
}

// Test_Main_Seeds_Payload_Content proves that fixed payload widths do not fix their bytes.
func Test_Main_Seeds_Payload_Content(t *testing.T) {
	t.Parallel()
	dotfiles := map[string]bool{}
	probes := map[string]bool{}
	fonts := map[string]bool{}
	for seed := uint64(0); seed < HARNESS_CORPUS; seed++ {
		dotfile, probe, font := harness_seeded_payloads(seed)
		dotfiles[dotfile] = true
		probes[probe] = true
		fonts[font] = true
	}
	for name, contents := range map[string]map[string]bool{
		"dotfile": dotfiles, "probe": probes, "font": fonts,
	} {
		if len(contents) != 4 {
			t.Errorf("the %s payload has %d contents, want 4", name, len(contents))
		}
	}
}

// The simulation cannot deliver an IO callback before its test root advances the Driver.
func Test_Main_Defers_Callback_Delivery(t *testing.T) {
	root := new_harness_root(HARNESS_FUZZ_SEED)
	first_retired := false
	second_retired := false
	harness_defer(root.Timeline, func() { first_retired = true })
	harness_defer_late(root.Timeline, func() { second_retired = true })
	if first_retired {
		t.Fatal("the harness delivered a callback during submission")
	}
	harness_arm_callback(root.Timeline)
	completed, drive_err := root.Driver.Run_Until(func() (finished bool) {
		return first_retired
	}, HARNESS_PUMP_DURATION_MAX)
	if drive_err != nil {
		t.Fatalf("the harness Driver failed: %v", drive_err)
	}
	if !completed {
		t.Fatal("the harness Driver did not deliver its callback")
	}
	if second_retired {
		t.Fatal("the harness delivered a late duplicate in the first pump pass")
	}
	harness_arm_callback(root.Timeline)
	completed, drive_err = root.Driver.Run_Until(func() (finished bool) {
		return second_retired
	}, HARNESS_PUMP_DURATION_MAX)
	if drive_err != nil {
		t.Fatalf("the harness Driver failed: %v", drive_err)
	}
	if !completed {
		t.Fatal("the harness Driver did not deliver its late callback")
	}
	root.Driver.Deinit()
}

// Test_Main_Selects_A_Feature_Subset proves that the normal corpus includes every fault.
func Test_Main_Selects_A_Feature_Subset(t *testing.T) {
	t.Parallel()
	covered := map[string]bool{}
	for seed := uint64(0); seed < HARNESS_CORPUS; seed++ {
		configuration := harness_configuration(seed)
		if configuration.Effective_User == 0 {
			continue
		}
		covered[configuration.Failure] = true
		if configuration.Failure == "" {
			continue
		}
		if configuration.Dotfile_Bytes != -1 {
			t.Fatalf(
				"the %q fault retained a %d-byte payload",
				configuration.Failure, configuration.Dotfile_Bytes)
		}
		if configuration.Path_Boundary != 0 {
			t.Fatalf(
				"the %q fault retained path-boundary feature %d",
				configuration.Failure, configuration.Path_Boundary)
		}
	}
	for _, failure := range []string{
		"", "direnv", "dotfiles", "defaults", "fonts", "nvim", "fzf", "maddox",
		"rust", "rust-link", "jj", "rg", "fd", "ghostty",
	} {
		if !covered[failure] {
			t.Errorf("the normal seed corpus does not cover the %q failure", failure)
		}
	}
}

// Test_Main_Selects_A_Fixture_Subset proves that no source fixture stays active in every run.
func Test_Main_Selects_A_Fixture_Subset(t *testing.T) {
	t.Parallel()
	covered := map[string]bool{}
	for seed := uint64(0); seed < HARNESS_CORPUS; seed++ {
		configuration := harness_configuration(seed)
		files, _ := simulation_files(seed, configuration)
		directories := simulation_directories(files, configuration)
		for name, present := range map[string]bool{
			"profile":            files["/profile"] != nil,
			"configuration":      files["/nested/config"] != nil,
			"ignored":            files["/e0/ignored"] != nil,
			"boundary-file":      files["/f0"] != nil,
			"boundary-directory": directories["/"+HARNESS_EMPTY_DIRECTORY],
			"io-witness":         files["/i0"] != nil,
			"nested-file":        files["/d0/x"] != nil,
		} {
			covered[fmt.Sprintf("%s/%t", name, present)] = true
		}
	}
	for _, name := range []string{
		"profile", "configuration", "ignored", "boundary-file",
		"boundary-directory", "io-witness", "nested-file",
	} {
		for _, present := range []bool{false, true} {
			feature := fmt.Sprintf("%s/%t", name, present)
			if !covered[feature] {
				t.Errorf(
					"the normal seed corpus does not cover fixture feature %q",
					feature)
			}
		}
	}
}

// Test_Main_Covers_Each_IO_Fault proves that the normal corpus reaches each shared IO result.
func Test_Main_Covers_Each_IO_Fault(t *testing.T) {
	t.Parallel()
	covered := map[string]bool{}
	targeted := map[string]bool{}
	recovered := map[string]bool{}
	for seed := uint64(0); seed < HARNESS_CORPUS; seed++ {
		configuration := harness_configuration(seed)
		if configuration.Effective_User == 0 {
			continue
		}
		covered[configuration.IO_Fault] = true
		if configuration.IO_Fault == "" {
			continue
		}
		targeted[configuration.IO_Fault+"/"+configuration.IO_Fault_Target] = true
		recovered[configuration.IO_Fault+"/"+configuration.IO_Fault_Target+"/"+
			configuration.IO_Fault_Site] = true
		if configuration.Failure != "" {
			t.Fatalf(
				"the %q IO fault retained the %q command fault",
				configuration.IO_Fault, configuration.Failure)
		}
		if configuration.Dotfile_Bytes != -1 {
			t.Fatalf(
				"the %q IO fault retained a %d-byte payload",
				configuration.IO_Fault, configuration.Dotfile_Bytes)
		}
		if configuration.Path_Boundary != 0 {
			t.Fatalf(
				"the %q IO fault retained path-boundary feature %d",
				configuration.IO_Fault, configuration.Path_Boundary)
		}
	}
	for _, fault := range []string{
		"", "status-error", "open-error", "create-error", "make-directory-error",
		"read-directory-error", "read-error", "read-did-not-retire",
		"read-retired-twice", "read-negative-count", "read-over-count",
		"write-error", "write-did-not-retire", "write-retired-twice",
		"write-zero-count", "write-negative-count", "write-over-count",
		"close-error", "close-did-not-retire", "close-retired-twice",
		"spawn-error", "spawn-did-not-retire", "spawn-retired-twice",
		"spawn-nonzero-exit",
	} {
		if !covered[fault] {
			t.Errorf("the normal seed corpus does not cover the %q IO fault", fault)
		}
	}
	for _, fault := range []string{
		"status-error", "open-error", "create-error", "make-directory-error",
		"read-error", "read-did-not-retire", "read-retired-twice",
		"read-negative-count", "read-over-count", "write-error",
		"write-did-not-retire", "write-retired-twice", "write-zero-count",
		"write-negative-count", "write-over-count", "close-error",
		"close-did-not-retire", "close-retired-twice",
	} {
		for _, target := range []string{"mirror", "font"} {
			pair := fault + "/" + target
			if !targeted[pair] {
				t.Errorf(
					"the normal seed corpus does not cover the %q "+
						"IO fault target", pair)
			}
		}
	}
	harness_assert_recoverable_sites(t, recovered)
	harness_assert_terminal_process_sites(t, recovered)
}

// Test_Main_Reaches_Production_Bounds proves that rare full-size configurations stay reachable.
func Test_Main_Reaches_Production_Bounds(t *testing.T) {
	t.Parallel()
	payload, file_path, directory_path := false, false, false
	one_byte_io, two_byte_io := false, false
	for seed := uint64(0); seed < HARNESS_CORPUS; seed++ {
		configuration := harness_configuration(seed)
		if configuration.Effective_User == 0 {
			continue
		}
		if configuration.Failure != "" {
			continue
		}
		if configuration.Dotfile_Bytes == 1048575 {
			payload = true
		}
		if configuration.Path_Boundary == 1 {
			file_path = true
		}
		if configuration.Path_Boundary == 2 {
			directory_path = true
		}
		if configuration.IO_Chunk_Bytes == 1 {
			one_byte_io = true
		}
		if configuration.IO_Chunk_Bytes == 2 {
			two_byte_io = true
		}
	}
	if !payload {
		t.Error("the normal seed corpus does not cover the full dotfile payload")
	}
	if !file_path {
		t.Error("the normal seed corpus does not cover the full relative file path")
	}
	if !directory_path {
		t.Error("the normal seed corpus does not cover the full relative directory path")
	}
	if !one_byte_io {
		t.Error("the normal seed corpus does not cover one-byte IO completions")
	}
	if !two_byte_io {
		t.Error("the normal seed corpus does not cover two-byte IO completions")
	}
}

// Test_Main sweeps the seed corpus, driving Main over each seed's filesystem and
// asserting every invariant holds; `go test -fuzz=Fuzz_Main` explores beyond the corpus.
func Test_Main(t *testing.T) {
	for seed := uint64(0); seed < HARNESS_CORPUS; seed++ {
		drive(t, seed)
	}
}

// The recoverable list remains independent from the production fault constants on purpose.
func harness_assert_recoverable_sites(t *testing.T, recovered map[string]bool) {
	t.Helper()
	for _, site := range []string{
		"status-error/mirror/probe", "status-error/font/probe",
		"spawn-error/mirror/probe",
		"spawn-retired-twice/mirror/probe", "spawn-nonzero-exit/mirror/probe",
	} {
		if !recovered[site] {
			t.Errorf("the normal seed corpus does not cover the %q IO site", site)
		}
	}
}

// The terminal list remains independent from the production fault constants on purpose.
func harness_assert_terminal_process_sites(t *testing.T, covered map[string]bool) {
	t.Helper()
	for _, site := range []string{
		"spawn-did-not-retire/mirror/required",
		"spawn-error/mirror/gitignore",
		"spawn-retired-twice/mirror/gitignore",
		"spawn-nonzero-exit/mirror/gitignore",
	} {
		if !covered[site] {
			t.Errorf("the normal seed corpus does not cover the %q IO site", site)
		}
	}
}
