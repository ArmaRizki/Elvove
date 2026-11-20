package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// galleryRequest builds the request the mux would hand to limitGallery, with the
// path values already filled in.
func galleryRequest(method, token, id string) *http.Request {
	path := "/api/gallery/" + token
	if id != "" {
		path += "/photos/" + id
	}
	request := httptest.NewRequest(method, path, nil)
	request.SetPathValue("token", token)
	if id != "" {
		request.SetPathValue("id", id)
	}
	return request
}

// countingHandler records how often the route behind the limiter ran, and
// answers with the given status so the caller's status shows whether the
// limiter refused before the route or the route itself answered.
func countingHandler(calls *int, status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		*calls++
		w.WriteHeader(status)
	}
}

func TestGalleryLimiterFollowsTheTokenNotTheAddress(t *testing.T) {
	server := &Server{Limiter: NewRateLimiter()}
	calls := 0
	route := server.limitGallery(countingHandler(&calls, http.StatusOK))

	// Two different visitors behind one address, each with their own token and
	// each asking for far less than the window allows.
	first, second := "aabbccddeeff00112233445566778899", "99887766554433221100ffeeddccbbaa"
	for i := 0; i < 10; i++ {
		for _, token := range []string{first, second} {
			recorder := httptest.NewRecorder()
			route(recorder, galleryRequest(http.MethodGet, token, ""))
			if recorder.Code != http.StatusOK {
				t.Fatalf("token %.8s: request %d answered %d, want 200", token, i+1, recorder.Code)
			}
		}
	}
	if calls != 20 {
		t.Fatalf("the route ran %d times, want 20", calls)
	}
}

func TestGalleryLimiterStopsOneTokenAtItsWindow(t *testing.T) {
	server := &Server{Limiter: NewRateLimiter()}
	calls := 0
	route := server.limitGallery(countingHandler(&calls, http.StatusOK))
	token := "aabbccddeeff00112233445566778899"

	for i := 0; i < galleryListPerMinute; i++ {
		recorder := httptest.NewRecorder()
		route(recorder, galleryRequest(http.MethodGet, token, ""))
		if recorder.Code != http.StatusOK {
			t.Fatalf("request %d answered %d, want 200", i+1, recorder.Code)
		}
	}
	recorder := httptest.NewRecorder()
	route(recorder, galleryRequest(http.MethodGet, token, ""))
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("request %d answered %d, want 429", galleryListPerMinute+1, recorder.Code)
	}
	if calls != galleryListPerMinute {
		t.Fatalf("the route ran %d times, want %d: a refused request must not reach it", calls, galleryListPerMinute)
	}
	// A refused listing does not spend the photo budget, and a different token
	// is untouched.
	recorder = httptest.NewRecorder()
	route(recorder, galleryRequest(http.MethodGet, token, "fileAAA111111"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("a photo for a throttled listing answered %d, want 200", recorder.Code)
	}
	recorder = httptest.NewRecorder()
	route(recorder, galleryRequest(http.MethodGet, "99887766554433221100ffeeddccbbaa", ""))
	if recorder.Code != http.StatusOK {
		t.Fatalf("another token answered %d, want 200", recorder.Code)
	}
}

func TestGalleryLimiterSeparatesReadsFromWrites(t *testing.T) {
	server := &Server{Limiter: NewRateLimiter()}
	calls := 0
	route := server.limitGallery(countingHandler(&calls, http.StatusOK))
	token := "aabbccddeeff00112233445566778899"

	for i := 0; i < galleryWritePerMinute; i++ {
		recorder := httptest.NewRecorder()
		route(recorder, galleryRequest(http.MethodPut, token, ""))
		if recorder.Code != http.StatusOK {
			t.Fatalf("save %d answered %d, want 200", i+1, recorder.Code)
		}
	}
	recorder := httptest.NewRecorder()
	route(recorder, galleryRequest(http.MethodPut, token, ""))
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("save %d answered %d, want 429", galleryWritePerMinute+1, recorder.Code)
	}
	// Saving does not close the page.
	recorder = httptest.NewRecorder()
	route(recorder, galleryRequest(http.MethodGet, token, ""))
	if recorder.Code != http.StatusOK {
		t.Fatalf("the listing answered %d after the save window closed, want 200", recorder.Code)
	}
}

// TestGalleryLimiterHasNoServerWithoutOne keeps the handler usable in tests and
// in a process that never built a limiter: it must pass the request through.
func TestGalleryLimiterHasNoServerWithoutOne(t *testing.T) {
	server := &Server{}
	calls := 0
	route := server.limitGallery(countingHandler(&calls, http.StatusOK))
	for i := 0; i < 400; i++ {
		recorder := httptest.NewRecorder()
		route(recorder, galleryRequest(http.MethodGet, "aabbccddeeff00112233445566778899", ""))
		if recorder.Code != http.StatusOK {
			t.Fatalf("request %d answered %d, want 200", i+1, recorder.Code)
		}
	}
	if calls != 400 {
		t.Fatalf("the route ran %d times, want 400", calls)
	}
}

// TestGalleryWindowsAreGenerousEnoughForOnePage documents the numbers: a client
// page for a folder of 352 photos, opened twice with a reload, must stay inside
// every window.
func TestGalleryWindowsAreGenerousEnoughForOnePage(t *testing.T) {
	const photos = 352
	if photos+2 > galleryPhotoPerMinute {
		t.Fatalf("one page asks for %d previews, the minute window allows %d", photos+2, galleryPhotoPerMinute)
	}
	if 2*photos > galleryPhotoPerHour {
		t.Fatalf("two pages ask for %d previews, the hour window allows %d", 2*photos, galleryPhotoPerHour)
	}
	if galleryListPerMinute < 10 || galleryListPerHour < 100 {
		t.Fatalf("the listing windows (%d/min, %d/h) are too small for a page that reloads",
			galleryListPerMinute, galleryListPerHour)
	}
	span := time.Minute
	if galleryPhotoPerMinute <= 0 || span <= 0 {
		t.Fatal("the windows must be positive")
	}
}

// TestGalleryLimitIsReachedThroughTheRealRouter answers the question the
// wrapper tests cannot: does a request to /api/gallery/{token}/photos reach
// limitGallery at all, or does an earlier pattern answer first?
//
// The mux matches on its own, so the path values only exist once net/http has
// routed the request. Driving limitGallery directly supplies them by hand and
// would keep passing even if no route ever called it.
func TestGalleryLimitIsReachedThroughTheRealRouter(t *testing.T) {
	server := &Server{Limiter: NewRateLimiter()}
	handler := server.Handler()
	const token = "aabbccddeeff00112233445566778899"
	path := "/api/gallery/" + token + "/photos"

	var last int
	var lastBody string
	for i := 0; i < galleryListPerMinute; i++ {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		last, lastBody = recorder.Code, recorder.Body.String()
		if last == http.StatusTooManyRequests {
			t.Fatalf("request %d was throttled before the window closed at %d", i+1, galleryListPerMinute)
		}
	}
	if last == http.StatusTooManyRequests {
		t.Fatalf("the last request inside the window answered 429")
	}
	t.Logf("inside the window: %d %s", last, lastBody)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("request %d through the router answered %d, want 429: the route does not pass through limitGallery",
			galleryListPerMinute+1, recorder.Code)
	}
}
