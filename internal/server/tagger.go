package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bogem/id3v2/v2"
	"github.com/go-flac/flacpicture"
	"github.com/go-flac/flacvorbis"
	"github.com/go-flac/go-flac"
)

// SongMeta holds metadata to embed into an audio file.
type SongMeta struct {
	Title    string
	Artist   string
	Album    string
	CoverURL string
}

func embedMetadata(ctx context.Context, filePath, fileType string, meta SongMeta, client *http.Client) error {
	switch strings.ToLower(fileType) {
	case "mp3":
		return embedMP3(ctx, filePath, meta, client)
	case "flac":
		return embedFLAC(ctx, filePath, meta, client)
	default:
		return nil
	}
}

func normalizeMIME(contentType string) string {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}
	switch ct {
	case "image/jpeg", "image/jpg", "image/pjpeg":
		return "image/jpeg"
	case "image/png", "image/x-png":
		return "image/png"
	case "":
		return "image/jpeg"
	default:
		return ct
	}
}

func fetchCover(ctx context.Context, coverURL string, client *http.Client) ([]byte, string, error) {
	if coverURL == "" {
		return nil, "", nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, coverURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("cover fetch failed: status=%d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	mime := normalizeMIME(resp.Header.Get("Content-Type"))
	return data, mime, nil
}

func embedMP3(ctx context.Context, filePath string, meta SongMeta, client *http.Client) error {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("open mp3 tag: %w", err)
	}
	defer tag.Close()

	tag.SetDefaultEncoding(id3v2.EncodingUTF8)
	tag.SetTitle(meta.Title)
	tag.SetArtist(meta.Artist)
	tag.SetAlbum(meta.Album)

	if meta.CoverURL != "" {
		coverData, mime, err := fetchCover(ctx, meta.CoverURL, client)
		if err == nil && len(coverData) > 0 {
			tag.AddAttachedPicture(id3v2.PictureFrame{
				Encoding:    id3v2.EncodingUTF8,
				MimeType:    mime,
				PictureType: id3v2.PTFrontCover,
				Description: "Cover",
				Picture:     coverData,
			})
		}
	}

	return tag.Save()
}

func embedFLAC(ctx context.Context, filePath string, meta SongMeta, client *http.Client) error {
	f, err := flac.ParseFile(filePath)
	if err != nil {
		return fmt.Errorf("parse flac: %w", err)
	}

	var cleaned []*flac.MetaDataBlock
	for _, m := range f.Meta {
		if m.Type != flac.VorbisComment && m.Type != flac.Picture {
			cleaned = append(cleaned, m)
		}
	}

	cmts := flacvorbis.New()
	cmts.Add(flacvorbis.FIELD_TITLE, meta.Title)
	cmts.Add(flacvorbis.FIELD_ARTIST, meta.Artist)
	cmts.Add(flacvorbis.FIELD_ALBUM, meta.Album)
	cmtBlock := cmts.Marshal()
	cleaned = append(cleaned, &cmtBlock)

	if meta.CoverURL != "" {
		coverData, mime, err := fetchCover(ctx, meta.CoverURL, client)
		if err != nil {
			return fmt.Errorf("fetch cover for flac: %w", err)
		}
		if len(coverData) > 0 {
			pic, err := flacpicture.NewFromImageData(
				flacpicture.PictureTypeFrontCover,
				"Cover",
				coverData,
				mime,
			)
			if err != nil {
				pic = &flacpicture.MetadataBlockPicture{
					PictureType: flacpicture.PictureTypeFrontCover,
					MIME:        mime,
					Description: "Cover",
					ImageData:   coverData,
				}
			}
			picBlock := pic.Marshal()
			cleaned = append(cleaned, &picBlock)
		}
	}

	f.Meta = cleaned
	return f.Save(filePath)
}
