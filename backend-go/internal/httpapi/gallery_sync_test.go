package httpapi

import (
	"context"
	"testing"
	"time"

	"github.com/elvove/photography/backend-go/internal/drive"
)

func testFiles() []drive.File {
	return []drive.File{
		{ID: "test-file-aaa", Name: "IMG_0001.jpg", Width: 6000, Height: 4000},
		{ID: "test-file-bbb", Name: "IMG_0002.jpg", Width: 4000, Height: 6000},
	}
}

// TestGalleryFingerprintFollowsTheListing pins what "unchanged" means for the
// photo rows: the same files with the same name and size skip the write;
// anything the rows store changes it.
func TestGalleryFingerprintFollowsTheListing(t *testing.T) {
	base := testFiles()
	if galleryFingerprint(base) != galleryFingerprint(testFiles()) {
		t.Fatal("the same listing must give the same fingerprint")
	}
	cases := map[string][]drive.File{
		"a photo removed":    base[:1],
		"a photo added":      append(testFiles(), drive.File{ID: "test-file-ccc", Name: "IMG_0003.jpg"}),
		"a photo renamed":    {{ID: "test-file-aaa", Name: "IMG_9999.jpg", Width: 6000, Height: 4000}, base[1]},
		"a photo re-encoded": {{ID: "test-file-aaa", Name: "IMG_0001.jpg", Width: 3000, Height: 2000}, base[1]},
		"a different folder": {{ID: "test-file-zzz", Name: "IMG_0001.jpg", Width: 6000, Height: 4000}, base[1]},
	}
	for name, files := range cases {
		if galleryFingerprint(files) == galleryFingerprint(base) {
			t.Errorf("%s: the fingerprint did not change, so the rows would not be updated", name)
		}
	}
}

// syncBookingForTest makes a booking row to hold the photos.
func syncBookingForTest(t *testing.T, server *Server, token, name string) galleryBooking {
	t.Helper()
	var id int64
	err := server.Store.Pool.QueryRow(context.Background(),
		// The columns the bookings table has required since the first schema:
		// a row without them is not a booking the API could have written.
		`INSERT INTO bookings(name, email, phone, event_date, terms_accepted, status,
		                      package_title, selection_token, raw_folder_id)
		 VALUES ($1,'sync@test.local','0800000000','2030-01-01',true,'PENDING',
		         'Test Package',$2,'test-folder') RETURNING id`, name, token).Scan(&id)
	if err != nil {
		t.Fatalf("insert booking: %v", err)
	}
	t.Cleanup(func() {
		_, _ = server.Store.Pool.Exec(context.Background(), `DELETE FROM bookings WHERE id=$1`, id)
	})
	return galleryBooking{ID: id, Name: name, RawFolderID: "test-folder"}
}

func countBookingPhotos(t *testing.T, server *Server, booking galleryBooking) int {
	t.Helper()
	var count int
	if err := server.Store.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM booking_photos WHERE booking_id=$1`, booking.ID).Scan(&count); err != nil {
		t.Fatalf("count photos: %v", err)
	}
	return count
}

// bookingPhotoRowsRead reads the stored rows while the statement waits a second:
// every column as text, ordered, which is what a page serialises. The wait is a
// WHERE clause, so the rows come back after it, and the second read then lands in
// a later transaction than any write made between the two reads.
const bookingPhotoRowsRead = `SELECT string_agg(t::text, '|' ORDER BY t.id)
	FROM (SELECT * FROM booking_photos WHERE booking_id=$1) t
	WHERE NOT pg_sleep(1.1)::text = 'x'`

// TestSyncBookingPhotosSkipsUnchangedRows runs the real statements against a
// real database: the rows are written once, and a second read of the same
// cached listing must not write them again. A fake would only test the fake.
func TestSyncBookingPhotosSkipsUnchangedRows(t *testing.T) {
	server := &Server{Store: testPool(t), galleryListed: map[string]galleryListing{}}
	booking := syncBookingForTest(t, server, "testtoken000000000000000000000001", "Sync One")
	ctx := context.Background()

	// A booking that shares nothing with the others starts out with no rows.
	if n := countBookingPhotos(t, server, booking); n != 0 {
		t.Fatalf("a fresh booking holds %d rows, want 0", n)
	}

	if _, err := server.syncBookingPhotos(ctx, booking, testFiles(), ""); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if n := countBookingPhotos(t, server, booking); n != 2 {
		t.Fatalf("%d rows after the first sync, want 2", n)
	}
	// A rename is copied into the row, so the listing is what the page shows.
	var stored string
	if err := server.Store.Pool.QueryRow(ctx,
		`SELECT file_name FROM booking_photos WHERE booking_id=$1 AND drive_file_id='test-file-aaa' AND NOT pg_sleep(1.1)::text='x'`, booking.ID).Scan(&stored); err != nil {
		t.Fatalf("read the row: %v", err)
	}
	if stored != "IMG_0001.jpg" {
		t.Fatalf("file_name = %q, want IMG_0001.jpg", stored)
	}

	// The same listing again: no write at all. The sync repeats its INSERT ...
	// ON CONFLICT DO UPDATE and its DELETE on every page load, so "unchanged"
	// has to mean the rows are not touched; updated_at is what shows a rewrite.
	//
	// The cache entry stands in for the one the first read stores: this test
	// calls the sync directly, and a folder with no cached listing is exactly
	// the case that must write.
	server.galleryListed[booking.RawFolderID] = galleryListing{files: testFiles(), at: time.Now(),
		synced: map[int64]string{booking.ID: galleryFingerprint(testFiles())}}
	var first, second string
	if err := server.Store.Pool.QueryRow(ctx, bookingPhotoRowsRead, booking.ID).Scan(&first); err != nil {
		t.Fatalf("read the rows: %v", err)
	}
	if first == "" {
		t.Fatal("the booking holds no rows to compare")
	}
	if first == "" {
		t.Fatal("the booking holds no rows to compare")
	}
	// The second sync runs in a later transaction than every write so far, so a
	// rewrite it made would show in the rows read after it.
	time.Sleep(1100 * time.Millisecond)
	if _, err := server.syncBookingPhotos(ctx, booking, testFiles(), server.syncedFor(booking)); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	if err := server.Store.Pool.QueryRow(ctx, bookingPhotoRowsRead, booking.ID).Scan(&second); err != nil {
		t.Fatalf("read the rows again: %v", err)
	}
	if first != second {
		t.Fatal("an unchanged listing rewrote the rows")
	}
	if n := countBookingPhotos(t, server, booking); n != 2 {
		t.Fatalf("%d rows after an unchanged sync, want 2", n)
	}
}

// TestSyncBookingPhotosUpdatesChangedRows is the other half: a folder that did
// change must reach the rows, including a picked photo that left the folder,
// which is kept.
func TestSyncBookingPhotosUpdatesChangedRows(t *testing.T) {
	server := &Server{Store: testPool(t), galleryListed: map[string]galleryListing{}}
	ctx := context.Background()
	booking := syncBookingForTest(t, server, "testtoken000000000000000000000002", "Sync Two")

	if _, err := server.syncBookingPhotos(ctx, booking, testFiles(), ""); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if _, err := server.Store.Pool.Exec(ctx,
		`UPDATE booking_photos SET selected=true WHERE booking_id=$1 AND drive_file_id='test-file-bbb'`, booking.ID); err != nil {
		t.Fatalf("mark a pick: %v", err)
	}

	changed := []drive.File{
		{ID: "test-file-aaa", Name: "IMG_0001-renamed.jpg", Width: 6000, Height: 4000},
		{ID: "test-file-ccc", Name: "IMG_0003.jpg", Width: 5000, Height: 5000},
	}
	if _, err := server.syncBookingPhotos(ctx, booking, changed, server.syncedFor(booking)); err != nil {
		t.Fatalf("sync after the folder changed: %v", err)
	}
	var name string
	if err := server.Store.Pool.QueryRow(ctx,
		`SELECT file_name FROM booking_photos WHERE booking_id=$1 AND drive_file_id='test-file-aaa'`, booking.ID).Scan(&name); err != nil {
		t.Fatalf("read the renamed row: %v", err)
	}
	if name != "IMG_0001-renamed.jpg" {
		t.Fatalf("file_name = %q, want the new name", name)
	}
	if n := countBookingPhotos(t, server, booking); n != 3 {
		t.Fatalf("%d rows, want 3: the new photo is added and the picked one is kept", n)
	}
	var picked bool
	if err := server.Store.Pool.QueryRow(ctx,
		`SELECT selected FROM booking_photos WHERE booking_id=$1 AND drive_file_id='test-file-bbb'`, booking.ID).Scan(&picked); err != nil {
		t.Fatalf("read the picked row: %v", err)
	}
	if !picked {
		t.Fatal("a photo the client picked was dropped when it left the folder")
	}
}
