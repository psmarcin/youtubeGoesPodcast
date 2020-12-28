package app

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kkdai/youtube"
	"github.com/kkdai/youtube/pkg/decipher"
	"go.opentelemetry.io/otel/label"
)

const (
	YoutubeVideoBaseURL = "https://www.youtube.com/watch?v="
)

type FileService struct{}

type Details struct {
	Url     url.URL
	Quality string
}

func NewFileService() FileService {
	return FileService{}
}

func (f FileService) GetDetails(ctx context.Context, videoId string) (Details, error) {
	details := Details{}
	ctx, span := tracer.Start(ctx, "file-get-details")
	span.SetAttributes(label.String("videoId", videoId))
	defer span.End()

	yt := youtube.NewYoutube(true, true)

	err := yt.DecodeURL(YoutubeVideoBaseURL + videoId)
	if err != nil {
		return details, err
	}

	audioStream, err := getAudioStream(yt.GetStreamInfo().Streams)
	if err != nil {
		return details, err
	}

	audioStreamRawUrl, err := getStreamUrl(videoId, audioStream)
	if err != nil {
		return details, err
	}

	audioStreamUrl, err := url.Parse(audioStreamRawUrl)
	if err != nil {
		return details, err
	}
	details.Url = *audioStreamUrl

	return details, nil
}

func getAudioStream(streams []youtube.Stream) (youtube.Stream, error) {
	for _, stream := range streams {
		if strings.HasPrefix(stream.MimeType, "audio") {
			return stream, nil
		}
	}

	if len(streams) > 0 {
		return streams[0], nil
	}

	return youtube.Stream{}, errors.New("can't find audio")
}

func getStreamUrl(videoId string, stream youtube.Stream) (string, error) {
	streamURL := stream.URL
	if streamURL == "" {
		cipher := stream.Cipher
		if cipher == "" {
			return "", errors.New("no cipher")
		}
		client := getHTTPClient()
		d := decipher.NewDecipher(client)
		decipherUrl, err := d.Url(videoId, cipher)
		if err != nil {
			return "", err
		}
		streamURL = decipherUrl
	}
	return streamURL, nil
}

func getHTTPClient() *http.Client {
	// setup a http client
	httpTransport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	httpClient := &http.Client{Transport: httpTransport}

	return httpClient
}
