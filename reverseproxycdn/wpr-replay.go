package reverseproxycdn

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/martian/v3/proxyutil"
	"github.com/google/martian/v3/reverseproxycdn/webpagereplay"
)

type WprReplay struct {
	wprArchiveFile string
	injectScripts  string
	transformers   []webpagereplay.ResponseTransformer
	quietMode      bool

	archive *webpagereplay.Archive
}

func (r *WprReplay) processInjectedScripts(timeSeedMs int64) error {
	if r.injectScripts != "" {
		for _, scriptFile := range strings.Split(r.injectScripts, ",") {
			log.Printf("Loading script from %v\n", scriptFile)
			// Replace {{WPR_TIME_SEED_TIMESTAMP}} with the time seed.
			replacements := map[string]string{"{{WPR_TIME_SEED_TIMESTAMP}}": strconv.FormatInt(timeSeedMs, 10)}
			si, err := webpagereplay.NewScriptInjectorFromFile(scriptFile, replacements)
			if err != nil {
				return fmt.Errorf("error opening script %s: %v", scriptFile, err)
			}
			r.transformers = append(r.transformers, si)
		}
	}

	return nil
}

// updateDate is the basic function for date adjustment.
func updateDate(h http.Header, name string, now, oldNow time.Time) {
	val := h.Get(name)
	if val == "" {
		return
	}
	oldTime, err := http.ParseTime(val)
	if err != nil {
		return
	}
	newTime := now.Add(oldTime.Sub(oldNow))
	h.Set(name, newTime.UTC().Format(http.TimeFormat))
}

// updateDates updates "Date" header as current time and adjusts "Last-Modified"/"Expires" against it.
func updateDates(h http.Header, now time.Time) {
	oldNow, err := http.ParseTime(h.Get("Date"))
	h.Set("Date", now.UTC().Format(http.TimeFormat))
	if err != nil {
		return
	}
	updateDate(h, "Last-Modified", now, oldNow)
	updateDate(h, "Expires", now, oldNow)
}

func (r *WprReplay) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	if req.URL.Path == "/web-page-replay-generate-200" {
		resp = proxyutil.NewResponse(http.StatusOK, nil, req)
		return
	}
	if req.URL.Path == "/web-page-replay-command-exit" {
		log.Printf("Shutting down. Received /web-page-replay-command-exit")
		os.Exit(0)
		return
	}
	if req.URL.Path == "/web-page-replay-reset-replay-chronology" {
		log.Printf("Received /web-page-replay-reset-replay-chronology")
		log.Printf("Reset replay order to start.")
		r.archive.StartNewReplaySession()
		return
	}
	// fixupRequestURL(req, proxy.scheme)

	// Lookup the response in the archive.
	_, storedResp, err := r.archive.FindRequest(req)
	if err != nil {
		log.Println("======== WprReplay: couldn't find matching request", req.URL, err)
		resp = proxyutil.NewResponse(http.StatusNotFound, nil, req)
		return
	}
	// defer storedResp.Body.Close()

	// Check if the stored Content-Encoding matches an encoding allowed by the client.
	// If not, transform the response body to match the client's Accept-Encoding.
	clientAE := strings.ToLower(req.Header.Get("Accept-Encoding"))
	originCE := strings.ToLower(storedResp.Header.Get("Content-Encoding"))
	if !strings.Contains(clientAE, originCE) {
		log.Printf("translating Content-Encoding [%s] -> [%s]", originCE, clientAE)
		body, err2 := ioutil.ReadAll(storedResp.Body)
		if err2 != nil {
			log.Printf("error reading response body from archive: %v", err)
			resp = proxyutil.NewResponse(http.StatusNotFound, nil, req)
			err = err2
			return
		}
		body, err = webpagereplay.DecompressBody(originCE, body)
		if err != nil {
			log.Printf("error decompressing response body: %v", err)
			resp = proxyutil.NewResponse(http.StatusNotFound, nil, req)
			return
		}
		body, ce, err2 := webpagereplay.CompressBody(clientAE, body)
		if err2 != nil {
			log.Printf("error recompressing response body: %v", err)
			resp = proxyutil.NewResponse(http.StatusNotFound, nil, req)
			return
		}
		storedResp.Header.Set("Content-Encoding", ce)
		storedResp.Body = ioutil.NopCloser(bytes.NewReader(body))
		// ContentLength has changed, so update the outgoing headers accordingly.
		if storedResp.ContentLength >= 0 {
			storedResp.ContentLength = int64(len(body))
			storedResp.Header.Set("Content-Length", strconv.Itoa(len(body)))
		}
	}

	// Update dates in response header.
	updateDates(storedResp.Header, time.Now())

	// Transform.
	for _, t := range r.transformers {
		t.Transform(req, storedResp)
	}

	// Forward the response.
	log.Printf("serving %v response", storedResp.StatusCode)
	resp = storedResp
	return
}

func NewWprReplay(wprArchiveFile, injectScripts string) *WprReplay {
	r := &WprReplay{
		wprArchiveFile: wprArchiveFile,
		injectScripts:  injectScripts,
	}
	var err error
	r.archive, err = webpagereplay.OpenArchive(r.wprArchiveFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening archive file: %v", err)
		os.Exit(1)
	}
	log.Printf("Opened archive %s", r.wprArchiveFile)
	// archive.ServeResponseInChronologicalSequence = r.serveResponseInChronologicalSequence
	// archive.DisableFuzzyURLMatching = r.disableFuzzyURLMatching
	// if archive.DisableFuzzyURLMatching {
	// 	log.Printf("Disabling fuzzy URL matching.")
	// }

	timeSeedMs := r.archive.DeterministicTimeSeedMs
	if timeSeedMs == 0 {
		// The time seed hasn't been set in the archive. Time seeds used to not be
		// stored in the archive, so this is expected to happen when loading old
		// archives. Just revert to the previous behavior: use the current time as
		// the seed.
		timeSeedMs = time.Now().Unix() * 1000
	}
	if err := r.processInjectedScripts(timeSeedMs); err != nil {
		log.Printf("Error processing injected scripts: %v", err)
		os.Exit(1)
	}
	return r
}
