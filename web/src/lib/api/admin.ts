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
	certificateEnabled: boolean;
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

/** Mirrors lessonhttp.userLessonActivityResponse: una lección que el
 * usuario abrió y/o completó. Las fechas pueden venir null. */
export type AdminUserLessonActivity = {
	courseSlug: string;
	courseTitle: string;
	lessonOrder: number;
	lessonTitle: string;
	firstViewedAt: string | null;
	lastViewedAt: string | null;
	completed: boolean;
	completedAt: string | null;
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

/** Mirrors adminLessonResponse — una "sesión". `classOrder` falta si no
 * está en ninguna clase. */
export type AdminLesson = {
	order: number;
	title: string;
	classOrder?: number;
	content: AdminLessonContent[];
};

/** Mirrors adminClassResponse. */
export type AdminClass = {
	order: number;
	title: string;
};

/** Mirrors adminCourseDetailResponse. */
export type AdminCourseDetail = AdminCourse & {
	classes: AdminClass[];
	lessons: AdminLesson[];
};

/** Todos los cursos, publicados y en borrador. */
export async function listAdminCourses(): Promise<AdminCourse[]> {
	return (await apiFetch<AdminCourse[] | null>("/admin/courses")) ?? [];
}

/** Un curso con toda su currícula (lecciones + contenido de cada una), para
 * armar la pantalla de edición de una sola pasada. Los arrays anidados se
 * normalizan acá (Go serializa un slice nil como null) para que los
 * componentes puedan hacer .map/.length sin guardas. */
export async function getAdminCourse(slug: string): Promise<AdminCourseDetail> {
	const course = await apiFetch<AdminCourseDetail>(`/admin/courses/${slug}`);
	return {
		...course,
		classes: course.classes ?? [],
		lessons: (course.lessons ?? []).map((lesson) => ({ ...lesson, content: lesson.content ?? [] })),
	};
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

/** Prende/apaga la emisión automática de certificado al completar este
 * curso al 100% (ver internal/certificate en el API). */
export function setCourseCertificateEnabled(slug: string, enabled: boolean): Promise<void> {
	return apiFetch<void>(`/admin/courses/${slug}/certificate`, { method: "PATCH", body: { enabled } });
}

/** Borra el curso completo (cascada: lecciones, contenido, inscripciones,
 * progreso — ver CourseRepo.Delete en el API). No hay vuelta atrás. */
export function deleteCourse(slug: string): Promise<void> {
	return apiFetch<void>(`/admin/courses/${slug}`, { method: "DELETE" });
}

export function createClass(slug: string, input: { title: string; order: number }): Promise<AdminClass> {
	return apiFetch(`/admin/courses/${slug}/classes`, { method: "POST", body: input });
}

export function updateClass(slug: string, order: number, title: string): Promise<AdminClass> {
	return apiFetch(`/admin/courses/${slug}/classes/${order}`, { method: "PUT", body: { title } });
}

/** Borra la clase. Sus sesiones (con su contenido y el progreso de los
 * alumnos) no se borran: quedan sin clase. */
export function deleteClass(slug: string, order: number): Promise<void> {
	return apiFetch(`/admin/courses/${slug}/classes/${order}`, { method: "DELETE" });
}

/** `classOrder` null = sin clase. */
export function createLesson(
	slug: string,
	input: { title: string; order: number; classOrder: number | null },
): Promise<{ order: number; title: string }> {
	return apiFetch(`/admin/courses/${slug}/lessons`, { method: "POST", body: input });
}

/** Renombra la sesión y la asigna a `classOrder` (null = sin clase). */
export function updateLesson(
	slug: string,
	order: number,
	input: { title: string; classOrder: number | null },
): Promise<{ order: number; title: string }> {
	return apiFetch(`/admin/courses/${slug}/lessons/${order}`, { method: "PUT", body: input });
}

/** Borra la sesión completa (cascada: su contenido y el progreso de los
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

export async function listAdminUsers(): Promise<AdminUser[]> {
	return (await apiFetch<AdminUser[] | null>("/admin/users")) ?? [];
}

/** Cambia el rol de un usuario. El propio API rechaza que un admin se
 * cambie el rol a sí mismo (ver AdminHandler.UpdateUserRole) — acá no se
 * repite esa validación, se confía en el 400 que devuelve. */
export function updateUserRole(id: number, role: UserRole): Promise<void> {
	return apiFetch<void>(`/admin/users/${id}`, { method: "PATCH", body: { role } });
}

/** Marca o desmarca la cuenta como verificada (ver
 * AdminHandler.UpdateUserVerification). */
export function setUserVerified(id: number, verified: boolean): Promise<void> {
	return apiFetch<void>(`/admin/users/${id}/verification`, { method: "PATCH", body: { verified } });
}

/** Lecciones que el usuario abrió o completó, ordenadas por curso. */
export async function listUserLessonActivity(id: number): Promise<AdminUserLessonActivity[]> {
	return (await apiFetch<AdminUserLessonActivity[] | null>(`/admin/users/${id}/lessons`)) ?? [];
}