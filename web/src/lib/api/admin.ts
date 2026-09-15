import { apiFetch, apiUpload } from "./client";
import type { UserRole } from "./auth";
import type { LessonContentType } from "./lessons";

/** Mirrors handlers.adminCourseResponse in the Go API. Unlike Course
 * (courses.ts), this includes draft courses and the fields only an admin
 * needs — GET /admin/courses and GET /admin/courses/{slug} both require
 * an ADMIN session (401/403 otherwise). */
export type AdminCourse = {
	id: number;
	title: string;
	slug: string;
	description: string | null;
	published: boolean;
	// Go marshals a nil/empty slice as null, not [] — always guard for it.
	visibleRoles: UserRole[] | null;
	createdAt: string;
};

export type AdminUser = {
	id: number;
	email: string;
	name: string;
	authProvider: string;
	emailVerified: boolean;
	role: UserRole;
	createdAt: string;
};

/** Mirrors handlers.lessonContentResponse. `order` is the stable id used to
 * edit/delete this piece — NOT its position in the array, which shifts as
 * soon as anything is deleted out of order (see the comment on the Go
 * struct). Only the field matching `type` comes populated. */
export type AdminLessonContent = {
	order: number;
	title: string;
	type: LessonContentType;
	youtubeUrl?: string;
	pdfUrl?: string;
	body?: string;
};

/** Mirrors handlers.adminLessonResponse. */
export type AdminLesson = {
	order: number;
	title: string;
	content: AdminLessonContent[];
};

/** Mirrors handlers.adminCourseDetailResponse. */
export type AdminCourseDetail = AdminCourse & {
	lessons: AdminLesson[];
};

/** Todos los cursos, publicados y en borrador. */
export function listAdminCourses(): Promise<AdminCourse[]> {
	return apiFetch<AdminCourse[]>("/admin/courses");
}

/** Un curso con toda su currícula (lecciones + contenido de cada una), para
 * armar la pantalla de edición de una sola pasada. */
export function getAdminCourse(slug: string): Promise<AdminCourseDetail> {
	return apiFetch<AdminCourseDetail>(`/admin/courses/${slug}`);
}

export function createCourse(input: { title: string; slug: string; description?: string | null }): Promise<AdminCourse> {
	return apiFetch<AdminCourse>("/admin/courses", { method: "POST", body: input });
}

/** Edita título/descripción. No toca el slug ni published — ver
 * setCoursePublished para eso. */
export function updateCourseDetails(slug: string, input: { title: string; description: string | null }): Promise<AdminCourse> {
	return apiFetch<AdminCourse>(`/admin/courses/${slug}`, { method: "PUT", body: input });
}

export function setCoursePublished(slug: string, published: boolean): Promise<void> {
	return apiFetch<void>(`/admin/courses/${slug}`, { method: "PATCH", body: { published } });
}

/** Borra el curso completo (cascada: lecciones, contenido, inscripciones,
 * progreso — ver CourseRepo.Delete en el API). No hay vuelta atrás. */
export function deleteCourse(slug: string): Promise<void> {
	return apiFetch<void>(`/admin/courses/${slug}`, { method: "DELETE" });
}

export function createLesson(slug: string, input: { title: string; order: number }): Promise<{ order: number; title: string }> {
	return apiFetch(`/admin/courses/${slug}/lessons`, { method: "POST", body: input });
}

export function updateLesson(slug: string, order: number, title: string): Promise<{ order: number; title: string }> {
	return apiFetch(`/admin/courses/${slug}/lessons/${order}`, { method: "PUT", body: { title } });
}

/** Borra la lección completa (cascada: su contenido y el progreso de los
 * usuarios sobre ella). No hay vuelta atrás. */
export function deleteLesson(slug: string, order: number): Promise<void> {
	return apiFetch(`/admin/courses/${slug}/lessons/${order}`, { method: "DELETE" });
}

/** Crea contenido de video o markdown. Para PDF ver uploadPDFContent — un
 * body JSON no es un buen lugar para mandar un archivo. */
export function createContent(
	slug: string,
	lessonOrder: number,
	input: { title: string; type: "video" | "markdown"; youtubeUrl?: string; body?: string },
): Promise<AdminLessonContent> {
	return apiFetch(`/admin/courses/${slug}/lessons/${lessonOrder}/content`, { method: "POST", body: input });
}

export function uploadPDFContent(slug: string, lessonOrder: number, title: string, file: File): Promise<AdminLessonContent> {
	const formData = new FormData();
	formData.append("title", title);
	formData.append("file", file);
	return apiUpload(`/admin/courses/${slug}/lessons/${lessonOrder}/content/pdf`, formData);
}

/** Edita el título y el campo propio del tipo (youtubeUrl o body). Ni el
 * tipo ni el archivo de un PDF ya subido son editables — para reemplazar
 * un PDF hay que borrar esta pieza y subir una nueva. */
export function updateContent(
	slug: string,
	lessonOrder: number,
	contentOrder: number,
	input: { title: string; youtubeUrl?: string; body?: string },
): Promise<AdminLessonContent> {
	return apiFetch(`/admin/courses/${slug}/lessons/${lessonOrder}/content/${contentOrder}`, { method: "PUT", body: input });
}

export function deleteContent(slug: string, lessonOrder: number, contentOrder: number): Promise<void> {
	return apiFetch(`/admin/courses/${slug}/lessons/${lessonOrder}/content/${contentOrder}`, { method: "DELETE" });
}

export function listAdminUsers(): Promise<AdminUser[]> {
	return apiFetch<AdminUser[]>("/admin/users");
}

/** Cambia el rol de un usuario. El propio API rechaza que un admin se
 * cambie el rol a sí mismo (ver AdminHandler.UpdateUserRole) — acá no se
 * repite esa validación, se confía en el 400 que devuelve. */
export function updateUserRole(id: number, role: UserRole): Promise<void> {
	return apiFetch<void>(`/admin/users/${id}`, { method: "PATCH", body: { role } });
}