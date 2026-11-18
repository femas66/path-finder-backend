// Package dataset menyimpan data statis yang dibutuhkan mesin rekomendasi.
//
// Berkas di-embed ke dalam binary agar server tidak bergantung pada path
// relatif saat dijalankan dari folder mana pun.
package dataset

import _ "embed"

// SkillsCSV adalah leksikon keahlian kanonik beserta aliasnya.
// Sumber: pathfinder-ml/data/raw/skills/skills.csv
//
//go:embed skills.csv
var SkillsCSV []byte

// JobsCSV adalah katalog lowongan awal yang di-seed ke tabel jobs.
// Kolom skill dipisahkan titik koma agar tidak bertabrakan dengan pemisah CSV.
//
//go:embed jobs.csv
var JobsCSV []byte
