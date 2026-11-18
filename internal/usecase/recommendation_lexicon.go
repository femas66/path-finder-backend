package usecase

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"

	"haji-koding/path-finder/internal/dataset"
)

const (
	// skillKindHard adalah tipe keahlian teknis pada leksikon.
	skillKindHard = "hard_skill"
	// skillKindSoft adalah tipe keahlian non-teknis pada leksikon.
	skillKindSoft = "soft_skill"
	// minMatchKeyLength mencegah pencocokan satu huruf seperti "C" atau "R"
	// yang terlalu sering muncul sebagai bagian kata biasa.
	minMatchKeyLength = 2
)

// skillEntry adalah satu baris leksikon keahlian.
type skillEntry struct {
	Canonical string
	Kind      string
	Category  string
}

// SkillProfile adalah hasil ekstraksi keahlian dari narasi pengguna.
type SkillProfile struct {
	HardSkills []string
	SoftSkills []string
	Interests  []string
}

// HasSkills bernilai true bila ada minimal satu hard atau soft skill.
func (p SkillProfile) HasSkills() bool {
	return len(p.HardSkills) > 0 || len(p.SoftSkills) > 0
}

// SkillLexicon mencocokkan kata pada narasi dengan nama keahlian kanonik.
//
// Pencocokan dilakukan per kata (bukan substring) supaya "Java" tidak ikut
// terdeteksi dari kata "JavaScript". Alias tetap didukung, sehingga "golang"
// dan "go lang" sama-sama menghasilkan "Go".
type SkillLexicon struct {
	// keys menyimpan urutan kunci sesuai urutan leksikon agar keluaran stabil.
	keys    []string
	entries map[string]skillEntry
}

// softSkillAliases memetakan frasa Indonesia ke nama kanonik soft skill.
// Dipakai sebagai pelengkap leksikon, karena bentuk kata pada narasi
// ("bekerja dalam tim") berbeda dari alias pada skills.csv ("kerja tim").
var softSkillAliases = []struct {
	Phrase    string
	Canonical string
}{
	{"bekerja dalam tim", "Teamwork"},
	{"bekerja sama", "Teamwork"},
	{"berkolaborasi", "Teamwork"},
	{"kerja kelompok", "Teamwork"},
	{"team work", "Teamwork"},
	{"berkomunikasi", "Communication"},
	{"kemampuan komunikasi", "Communication"},
	{"public speaking", "Communication"},
	{"memecahkan masalah", "Problem Solving"},
	{"pemecahan masalah", "Problem Solving"},
	{"berpikir kritis", "Critical Thinking"},
	{"berpikir analitis", "Analytical Thinking"},
	{"teliti", "Attention to Detail"},
	{"manajemen waktu", "Time Management"},
	{"mengatur waktu", "Time Management"},
	{"beradaptasi", "Adaptability"},
	{"cepat beradaptasi", "Adaptability"},
	{"belajar hal baru", "Continuous Learning"},
	{"mempelajari hal baru", "Continuous Learning"},
	{"cepat belajar", "Continuous Learning"},
	{"berinisiatif", "Initiative"},
	{"inisiatif", "Initiative"},
	{"memimpin", "Leadership"},
	{"empati", "Empathy"},
}

// NewSkillLexicon membangun leksikon dari isi skills.csv.
func NewSkillLexicon(csvData []byte) (*SkillLexicon, error) {
	reader := csv.NewReader(bytes.NewReader(csvData))
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("gagal membaca leksikon keahlian: %w", err)
	}

	lexicon := &SkillLexicon{entries: make(map[string]skillEntry)}
	for index, record := range records {
		// Baris pertama adalah header.
		if index == 0 || len(record) < 5 {
			continue
		}

		canonical := strings.TrimSpace(record[1])
		kind := strings.TrimSpace(record[2])
		category := strings.TrimSpace(record[3])
		if canonical == "" {
			continue
		}

		entry := skillEntry{Canonical: canonical, Kind: kind, Category: category}

		// Nama kanonik itu sendiri juga bisa muncul di narasi.
		lexicon.addKey(canonical, entry)

		var aliases []string
		if raw := strings.TrimSpace(record[4]); raw != "" {
			if err := json.Unmarshal([]byte(raw), &aliases); err != nil {
				// Alias rusak tidak boleh menggagalkan seluruh proses.
				aliases = nil
			}
		}
		for _, alias := range aliases {
			lexicon.addKey(alias, entry)
		}
	}

	// Tambahkan frasa soft skill khas narasi Indonesia.
	for _, alias := range softSkillAliases {
		lexicon.addKey(alias.Phrase, skillEntry{
			Canonical: alias.Canonical,
			Kind:      skillKindSoft,
		})
	}

	return lexicon, nil
}

func (l *SkillLexicon) addKey(raw string, entry skillEntry) {
	key := normalizeForMatch(raw)
	if len(key) < minMatchKeyLength {
		return
	}
	if _, exists := l.entries[key]; exists {
		return
	}
	l.entries[key] = entry
	l.keys = append(l.keys, key)
}

// Extract mengambil keahlian yang disebut pada narasi.
func (l *SkillLexicon) Extract(narrative string) SkillProfile {
	padded := " " + normalizeForMatch(narrative) + " "
	profile := SkillProfile{}

	seen := make(map[string]bool)
	for _, key := range l.keys {
		if !strings.Contains(padded, " "+key+" ") {
			continue
		}

		entry := l.entries[key]
		if seen[entry.Canonical] {
			continue
		}
		seen[entry.Canonical] = true

		switch entry.Kind {
		case skillKindHard:
			profile.HardSkills = append(profile.HardSkills, entry.Canonical)
		case skillKindSoft:
			profile.SoftSkills = append(profile.SoftSkills, entry.Canonical)
		default:
			profile.Interests = append(profile.Interests, entry.Canonical)
		}
	}

	return profile
}

// NormalizeSkillName memetakan satu nama keahlian ke bentuk kanoniknya.
// Bila tidak dikenali, nama asli dikembalikan apa adanya.
func (l *SkillLexicon) NormalizeSkillName(raw string) string {
	key := normalizeForMatch(raw)
	if entry, ok := l.entries[key]; ok {
		return entry.Canonical
	}
	return strings.TrimSpace(raw)
}

// NormalizeSkillList merapikan daftar keahlian dan membuang duplikat.
func (l *SkillLexicon) NormalizeSkillList(raw []string) []string {
	normalized := make([]string, 0, len(raw))
	seen := make(map[string]bool)

	for _, item := range raw {
		name := l.NormalizeSkillName(item)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if seen[key] {
			continue
		}
		seen[key] = true
		normalized = append(normalized, name)
	}

	return normalized
}

// DefaultSkillLexicon memakai data yang di-embed ke dalam binary.
func DefaultSkillLexicon() (*SkillLexicon, error) {
	return NewSkillLexicon(dataset.SkillsCSV)
}

// normalizeForMatch menyeragamkan teks agar pencocokan per kata konsisten.
//
// Hanya huruf, angka, serta karakter teknis + # . yang dipertahankan,
// karena karakter tersebut bagian dari nama keahlian seperti "C++", "C#",
// ".NET", dan "Node.js". Karakter lain dianggap pemisah kata.
//
// Titik di akhir kata dibuang agar "PostgreSQL." pada akhir kalimat tetap
// cocok dengan "PostgreSQL", sementara titik di tengah seperti "Node.js"
// dan di awal seperti ".NET" tetap dipertahankan.
func normalizeForMatch(text string) string {
	lowered := strings.ToLower(text)

	tokens := make([]string, 0, 32)
	var current strings.Builder

	flush := func() {
		if current.Len() == 0 {
			return
		}
		token := strings.TrimRight(current.String(), ".")
		current.Reset()
		if token == "" {
			return
		}
		tokens = append(tokens, token)
	}

	for _, char := range lowered {
		isWord := (char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '+' || char == '#' || char == '.'

		if isWord {
			current.WriteRune(char)
			continue
		}

		flush()
	}
	flush()

	return strings.Join(tokens, " ")
}
