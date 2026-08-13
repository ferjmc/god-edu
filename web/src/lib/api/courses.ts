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
