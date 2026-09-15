package handlers

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/ferjmc/god-edu/api/db"
)

func TestRequireNonEmpty(t *testing.T) {
	w := httptest.NewRecorder()
	if _, ok := requireNonEmpty(w, "   ", "título"); ok {
		t.Fatal("esperaba ok=false para un valor vacío")
	}
	if w.Code != 400 {
		t.Fatalf("esperaba 400, recibí %d", w.Code)
	}

	w2 := httptest.NewRecorder()
	value, ok := requireNonEmpty(w2, "  hola  ", "título")
	if !ok || value != "hola" {
		t.Fatalf("esperaba ok=true y value recortado, recibí ok=%v value=%q", ok, value)
	}
}

func TestHandleNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	if !handleNotFound(w, db.ErrNotFound, "no encontrado") {
		t.Fatal("esperaba true para db.ErrNotFound")
	}
	if w.Code != 404 {
		t.Fatalf("esperaba 404, recibí %d", w.Code)
	}

	w2 := httptest.NewRecorder()
	if handleNotFound(w2, errors.New("otro error"), "no encontrado") {
		t.Fatal("esperaba false para un error que no es db.ErrNotFound")
	}
	if w2.Code != 200 {
		t.Fatalf("esperaba que no escriba nada (200 por default), recibí %d", w2.Code)
	}
}
