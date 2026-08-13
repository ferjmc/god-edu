import { apiFetch } from "./client";

/** Mirrors lessonSummaryResponse in the Go API. */
export type LessonSummary = {
	order: number;
	title: string;
	completed: boolean;
};

export type LessonContentType = "video" | "pdf" | "markdown";

/** Mirrors lessonContentResponse. Only the field matching `type` is
 * populated — the API omits the other two (omitempty), not just leaves
 * them null. */
export type LessonContentItem = {
	title: string;
	type: LessonContentType;
	youtubeUrl?: string;
	pdfUrl?: string;
	body?: string;
};

/** Mirrors lessonDetailResponse. */
export type LessonDetail = {
	order: number;
	title: string;
	completed: boolean;
	content: LessonContentItem[];
};

/** Currícula completa de un curso, en orden. Requiere sesión — mismo
 * gating que getCourse (401 anónimo, 403 sin rol habilitado). */
export function listLessons(courseSlug: string): Promise<LessonSummary[]> {
	return apiFetch<LessonSummary[]>(`/courses/${courseSlug}/lessons`);
}

/** Contenido de una lección puntual, identificada por su orden dentro del
 * curso (no por id interno). No hay bloqueo secuencial: cualquier lección
 * de un curso visible se puede pedir directamente. */
export function getLesson(courseSlug: string, order: number): Promise<LessonDetail> {
	return apiFetch<LessonDetail>(`/courses/${courseSlug}/lessons/${order}`);
}

/** Marca la lección como completada para el usuario actual. Manual (botón),
 * no automático al entrar — ver nota en LessonHandler.MarkComplete del API
 * sobre por qué. Idempotente: tocarlo de nuevo no rompe nada. */
export function markLessonComplete(courseSlug: string, order: number): Promise<{ completed: boolean }> {
	return apiFetch<{ completed: boolean }>(`/courses/${courseSlug}/lessons/${order}/complete`, { method: "POST" });
}
