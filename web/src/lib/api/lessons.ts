import { apiFetch } from "./client";

/** Mirrors summaryResponse in the Go API. Una "sesión" de la currícula;
 * `classOrder`/`classTitle` faltan si no está asignada a ninguna clase. */
export type LessonSummary = {
	order: number;
	title: string;
	completed: boolean;
	classOrder?: number;
	classTitle?: string;
};

/** Sesiones agrupadas bajo una clase. `title` es null para el grupo de
 * sesiones sin clase. */
export type ClassGroup = {
	classOrder: number | null;
	title: string | null;
	lessons: LessonSummary[];
};

/** Agrupa la currícula (lista plana en orden de sesión) por clase, con las
 * clases en su propio orden y las sesiones sin clase al final. Si ninguna
 * sesión tiene clase devuelve un solo grupo sin título, que se muestra como
 * la lista plana de siempre. */
export function groupByClass(lessons: LessonSummary[]): ClassGroup[] {
	const byClass = new Map<number, ClassGroup>();
	const ungrouped: LessonSummary[] = [];
	for (const lesson of lessons) {
		if (lesson.classOrder === undefined) {
			ungrouped.push(lesson);
			continue;
		}
		let group = byClass.get(lesson.classOrder);
		if (!group) {
			group = { classOrder: lesson.classOrder, title: lesson.classTitle ?? null, lessons: [] };
			byClass.set(lesson.classOrder, group);
		}
		group.lessons.push(lesson);
	}
	const groups = [...byClass.values()].sort((a, b) => (a.classOrder ?? 0) - (b.classOrder ?? 0));
	if (ungrouped.length > 0) groups.push({ classOrder: null, title: null, lessons: ungrouped });
	return groups;
}

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
	classTitle?: string;
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
