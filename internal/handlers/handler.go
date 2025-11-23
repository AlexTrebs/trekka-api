package handlers

import "trekka-api/internal/services"

type Handler struct {
	imageService *services.ImageService
}

func New(imageService *services.ImageService) *Handler {
	return &Handler{
		imageService: imageService,
	}
}
