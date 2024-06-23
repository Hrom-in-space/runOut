package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"gorm.io/gorm"

	"runout/internal/domain"
	"runout/internal/models"
	"runout/internal/utils"
	"runout/pkg/logger"
)

func AddNeed(audioCh chan<- domain.Audio) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		log := logger.FromCtx(req.Context())
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

		if len(data) == 0 {
			log.Error("empty data")
			writer.WriteHeader(http.StatusBadRequest)

			return
		}

		audioCh <- domain.Audio{
			Data:   data,
			Format: format,
		}
		log.Info("Audio added", slog.String("format", format))

		writer.WriteHeader(http.StatusOK)
	}
}

func ListNeeds(db *gorm.DB) http.HandlerFunc {
	return func(respWriter http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		log := logger.FromCtx(ctx)

		var needs []models.Need
		result := db.Find(&needs)
		if result.Error != nil {
			log.Error("ListNeeds", logger.Error(result.Error))
			respWriter.WriteHeader(http.StatusInternalServerError)

			return
		}

		respWriter.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(respWriter).Encode(needs)
		if err != nil {
			log.Error("Encode needs", logger.Error(err))
			respWriter.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func ClearNeeds(db *gorm.DB) http.HandlerFunc {
	return func(respWriter http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		log := logger.FromCtx(ctx)

		result := db.Where("id > ?", "0").Delete(&models.Need{})
		if result.Error != nil {
			log.Error("ClearNeeds", logger.Error(result.Error))
			respWriter.WriteHeader(http.StatusInternalServerError)

			return
		}

		respWriter.WriteHeader(http.StatusOK)
	}
}

func DeleteOne(db *gorm.DB) http.HandlerFunc {
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

		result := db.Delete(&models.Need{ID: needID})
		if result.Error != nil {
			log.Error("ClearNeeds", logger.Error(result.Error))
			respWriter.WriteHeader(http.StatusInternalServerError)

			return
		}

		respWriter.WriteHeader(http.StatusOK)
	}
}
