import { apiFetch } from "./client";

/** Mirrors handlers.courseResponse in the Go API. */
export type Course = {
	id: number;
	title: string;
	slug: string;
	description: string | null;
	certificateEnabled: boolean;
	/** Si el usuario autenticado ya se inscribió en este curso (ver
	 * enrollInCourse). No decide si puede VER esta respuesta — eso lo
	 * decide el rol — solo si el frontend debe mostrar el panel completo
	 * de lecciones o el resumen con el botón "Inscribirme". Ausente en
	 * listCourses (ahí no hay un usuario en particular). */
	enrolled: boolean;
};

/** Published courses only — the API never returns drafts on this endpoint. */
export function listCourses(): Promise<Course[]> {
	return apiFetch<Course[]>("/courses");
}

/** Course detail. Requires a session — the API returns 401 for anonymous
 * requests, on purpose: course content isn't public, only the listing is. */
export function getCourse(slug: string): Promise<Course> {
	return apiFetch<Course>(`/courses/${slug}`);
}

/** Inscribe al usuario autenticado en el curso — acción explícita (botón
 * "Inscribirme" en el detalle), nunca automática. Idempotente del lado del
 * API: llamarla de nuevo no crea una segunda inscripción. Devuelve el curso
 * actualizado (enrolled: true) para no forzar un segundo round-trip. */
export function enrollInCourse(slug: string): Promise<Course> {
	return apiFetch<Course>(`/courses/${slug}/enroll`, { method: "POST" });
}

/** Mirrors handlers.courseProgressResponse en el backend. */
export type CourseProgress = {
	id: number;
	title: string;
	slug: string;
	description: string | null;
	certificateEnabled: boolean;
	totalLessons: number;
	completedLessons: number;
	progressPercent: number;
	/** order_index de la próxima lección pendiente; ausente si no tiene
	 * lecciones o ya las completó todas. */
	nextLessonOrder?: number;
	/** Fecha de inscripción; ausente si el usuario todavía no se inscribió
	 * en este curso. */
	enrolledAt?: string;
	/** Código del certificado emitido; ausente si el curso no otorga
	 * certificado o todavía no se completó al 100%. */
	certificateCode?: string;
};

/** Cursos visibles para el rol del usuario actual, con su progreso. Sin
 * inscripción explícita: "mis cursos" = todo lo que Course.VisibleTo deja
 * pasar para ese rol. */
export function listMyCourses(): Promise<CourseProgress[]> {
	return apiFetch<CourseProgress[]>("/courses/mine");
}
