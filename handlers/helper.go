package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/nilsmagnus/grib/griblib"
)

type WindHeader struct {
	RefTime string  `json:"refTime"`
	Nx      uint32  `json:"nx"`
	Ny      uint32  `json:"ny"`
	Lo1     float64 `json:"lo1"`
	La1     float64 `json:"la1"`
	Dx      float64 `json:"dx"`
	Dy      float64 `json:"dy"`
}

type WindResponse struct {
	Header WindHeader `json:"header"`
	U      []float64  `json:"u"`
	V      []float64  `json:"v"`
}

func DownloadMet(date string, metDir string) (string, error) {
	return downloadMetCycle(date, "00", metDir)
}

func downloadMetCycle(date, cycle, metDir string) (string, error) {
	filePath := filepath.Join(metDir, date+cycle)
	if fi, err := os.Stat(filePath); err == nil && fi.Size() > 0 {
		return filePath, nil
	}
	if err := os.MkdirAll(metDir, 0o755); err != nil {
		return "", err
	}
	template := "https://nomads.ncep.noaa.gov/cgi-bin/filter_gdas_0p25.pl?dir=/gdas.%s/%s/atmos&file=gdas.t%sz.pgrb2.0p25.f000&var_UGRD=on&var_VGRD=on&lev_10_m_above_ground=on"
	url := fmt.Sprintf(template, date, cycle, cycle)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad request: %s", resp.Status)
	}
	out, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		os.Remove(filePath)
		return "", err
	}
	return filePath, nil
}

// latestCycles returns GDAS cycles (00/06/12/18z) from `now` backwards,
// newest first. Upstream publishes with ~3-6h lag, so the newest
// candidates may 404; callers walk back until one downloads and parses.
func latestCycles(now time.Time, count int) []struct{ Date, Cycle string } {
	now = now.UTC()
	base := time.Date(now.Year(), now.Month(), now.Day(), (now.Hour()/6)*6, 0, 0, 0, time.UTC)
	out := make([]struct{ Date, Cycle string }, 0, count)
	for i := range count {
		t := base.Add(time.Duration(-i*6) * time.Hour)
		out = append(out, struct{ Date, Cycle string }{
			Date:  t.Format("20060102"),
			Cycle: fmt.Sprintf("%02d", t.Hour()),
		})
	}
	return out
}

func ExtractUVFromGrib(gribPath string) (WindHeader, []float64, []float64, error) {
	file, err := os.Open(gribPath)
	if err != nil {
		return WindHeader{}, nil, nil, err
	}
	defer file.Close()

	messages, err := griblib.ReadMessages(file)
	if err != nil {
		return WindHeader{}, nil, nil, err
	}

	var header WindHeader
	var uData, vData []float64
	for _, msg := range messages {
		if header.Nx == 0 {
			if g, ok := msg.Section3.Definition.(*griblib.Grid0); ok {
				header.Nx = g.Ni
				header.Ny = g.Nj
				header.Lo1 = float64(g.Lo1) / 1e6
				header.La1 = float64(g.La1) / 1e6
				header.Dx = float64(g.Di) / 1e6
				header.Dy = float64(g.Dj) / 1e6
			}
		}
		switch msg.Section4.ProductDefinitionTemplate.ParameterNumber {
		case 2:
			uData = msg.Section7.Data
		case 3:
			vData = msg.Section7.Data
		}
	}
	return header, uData, vData, nil
}

func ServeWeather(metDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		date := r.PathValue("date")
		if date == "" {
			http.Error(w, "missing date", http.StatusBadRequest)
			return
		}
		path, err := DownloadMet(date, metDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		header, u, v, err := ExtractUVFromGrib(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if u == nil || v == nil {
			http.Error(w, "U/V components missing from GRIB", http.StatusInternalServerError)
			return
		}
		header.RefTime = date
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		json.NewEncoder(w).Encode(WindResponse{Header: header, U: u, V: v})
	}
}

func ServeCurrentWeather(metDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, c := range latestCycles(time.Now(), 8) {
			path, err := downloadMetCycle(c.Date, c.Cycle, metDir)
			if err != nil {
				continue
			}
			header, u, v, err := ExtractUVFromGrib(path)
			if err != nil || u == nil || v == nil {
				os.Remove(path)
				continue
			}
			header.RefTime = c.Date + c.Cycle
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "public, max-age=600")
			json.NewEncoder(w).Encode(WindResponse{Header: header, U: u, V: v})
			return
		}
		http.Error(w, "no recent cycle available", http.StatusServiceUnavailable)
	}
}
