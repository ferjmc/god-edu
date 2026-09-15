import { apiFetch } from "./client";

/** Mirrors handlers.courseResponse in the Go API. */
export type Course = {
	id: number;
	title: string;
	slug: string;
	description: string | null;
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

/** Mirrors handlers.courseProgressResponse en el backend. */
export type CourseProgress = {
	id: number;
	title: string;
	slug: string;
	description: string | null;
	totalLessons: number;
	completedLessons: number;
	progressPercent: number;
	/** order_index de la próxima lección pendiente; ausente si no tiene
	 * lecciones o ya las completó todas. */
	nextLessonOrder?: number;
};

/** Cursos visibles para el rol del usuario actual, con su progreso. Sin
 * inscripción explícita: "mis cursos" = todo lo que Course.VisibleTo deja
 * pasar para ese rol. */
export function listMyCourses(): Promise<CourseProgress[]> {
	return apiFetch<CourseProgress[]>("/courses/mine");
}
