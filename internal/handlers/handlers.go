package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"runout/internal/domain"
	"runout/internal/utils"
	"runout/pkg/logger"
	"runout/pkg/pg"
)

func AddNeed(audioCh chan<- domain.Audio) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		log := logger.FromCtx(req.Context())
		// enableCors(&w)
		// TODO: security check real file type https://github.com/h2non/filetype
		audioFormats := []string{"flac", "m4a", "mp3", "mp4", "mpeg", "mpga", "oga", "ogg", "wav", "webm"}

		contentType := req.Header.Get("Content-Type")
		format := strings.Replace(contentType, "audio/", "", 1)
		if !utils.InSlice(audioFormats, format) {
			log.Error("wrong Content-Type", slog.String("content-type", contentType))
			writer.WriteHeader(http.StatusBadRequest)
			// TODO: return information about error
			return
		}

		data, err := io.ReadAll(req.Body)
		if err != nil {
			log.Error("ReadAll", logger.Error(err))
			writer.WriteHeader(http.StatusInternalServerError)

			return
		}
		defer req.Body.Close()

		audioCh <- domain.Audio{
			Data:   data,
			Format: format,
		}
		log.Info("Audio added", slog.String("format", format))

		writer.WriteHeader(http.StatusOK)
	}
}

type NeedLister interface {
	ListNeeds(ctx context.Context) ([]domain.Need, error)
}

func ListNeeds(trm pg.Manager, repo NeedLister) http.HandlerFunc {
	return func(respWriter http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		log := logger.FromCtx(ctx)

		var err error
		var needs []domain.Need
		if err := trm.Do(ctx, func(ctx context.Context) error {
			needs, err = repo.ListNeeds(ctx)
			if err != nil {
				return fmt.Errorf("error getting needs: %w", err)
			}

			return nil
		}); err != nil {
			log.Error("ListNeeds", logger.Error(err))
			respWriter.WriteHeader(http.StatusInternalServerError)
		}

		respWriter.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(respWriter).Encode(needs)
		if err != nil {
			log.Error("Encode needs", logger.Error(err))
			respWriter.WriteHeader(http.StatusInternalServerError)
		}
	}
}

type NeedsCleaner interface {
	ClearNeeds(ctx context.Context) error
}

func ClearNeeds(trm pg.Manager, repo NeedsCleaner) http.HandlerFunc {
	return func(respWriter http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		log := logger.FromCtx(ctx)

		var err error
		if err := trm.Do(ctx, func(ctx context.Context) error {
			err = repo.ClearNeeds(ctx)
			if err != nil {
				return fmt.Errorf("error getting needs: %w", err)
			}

			return nil
		}); err != nil {
			log.Error("ClearNeeds", logger.Error(err))
			respWriter.WriteHeader(http.StatusInternalServerError)
		}

		respWriter.WriteHeader(http.StatusOK)
	}
}

type NeedDeleter interface {
	DeleteOne(ctx context.Context, id int) error
}

func DeleteOne(trm pg.Manager, repo NeedDeleter) http.HandlerFunc {
	return func(respWriter http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		log := logger.FromCtx(ctx)
		rawID := mux.Vars(req)["id"]
		if rawID == "" {
			respWriter.WriteHeader(http.StatusNotFound)
			return
		}
		needID, err := strconv.Atoi(rawID)
		if err != nil {
			respWriter.WriteHeader(http.StatusNotFound)
			return
		}

		if err := trm.Do(ctx, func(ctx context.Context) error {
			err = repo.DeleteOne(ctx, needID)
			if err != nil {
				return fmt.Errorf("error getting needs: %w", err)
			}

			return nil
		}); err != nil {
			log.Error("ClearNeeds", logger.Error(err))
			respWriter.WriteHeader(http.StatusInternalServerError)
		}

		respWriter.WriteHeader(http.StatusOK)
	}
}
