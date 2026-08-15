import { useEffect, useState } from "react";
import { useStore } from "@nanostores/react";
import { $auth, initAuth } from "../../stores/auth";
import { listAdminCourses, type AdminCourse } from "../../lib/api/admin";
import { ApiError } from "../../lib/api/client";
import NewCourseForm from "./NewCourseForm";

/** Gates the admin course list behind session + ADMIN role. Same pattern as
 * CourseDetail: the page (admin/cursos.astro) is a static shell with no
 * data of its own, this island resolves auth and fetches everything
 * client-side — the HTML never contains admin data for anyone else. */
export default function AdminCoursesList() {
	const auth = useStore($auth);
	const [courses, setCourses] = useState<AdminCourse[] | null>(null);
	const [error, setError] = useState<string | null>(null);

	useEffect(() => {
		if (auth.status === "loading") void initAuth();
	}, [auth.status]);

	useEffect(() => {
		if (auth.status !== "authenticated" || auth.user.role !== "ADMIN") return;
		listAdminCourses()
			.then(setCourses)
			.catch((err) => setError(err instanceof ApiError ? err.message : "No pudimos cargar los cursos."));
	}, [auth.status]);

	if (auth.status === "loading") {
		return <span className="loading loading-spinner loading-md text-marian-blue" aria-label="Cargando" />;
	}

	if (auth.status === "anonymous") {
		return (
			<div className="flex flex-col items-start gap-4 rounded-[4px] border border-ink-15 bg-paper p-8">
				<p className="font-serif text-ink-80">Iniciá sesión con una cuenta de administrador para ver esto.</p>
				<a href="/ingresar" className="btn btn-cta">
					Ingresar
				</a>
			</div>
		);
	}

	if (auth.user.role !== "ADMIN") {
		return (
			<p role="alert" className="font-ui text-sm text-sacred-red">
				Esta sección es solo para administradores.
			</p>
		);
	}

	return (
		<div className="flex flex-col gap-6">
			<NewCourseForm onCreated={(slug) => window.location.assign(`/admin/cursos/${slug}`)} />

			{error ? (
				<p role="alert" className="font-ui text-sm text-sacred-red">
					{error}
				</p>
			) : !courses ? (
				<span className="loading loading-spinner loading-md text-marian-blue" aria-label="Cargando cursos" />
			) : courses.length === 0 ? (
				<p className="font-ui text-sm text-ink-60">Todavía no hay cursos cargados.</p>
			) : (
				<table className="w-full border-collapse font-ui text-sm">
					<thead>
						<tr className="border-b border-ink-15 text-left text-ink-60">
							<th className="py-2 pr-4 font-medium">Título</th>
							<th className="py-2 pr-4 font-medium">Slug</th>
							<th className="py-2 pr-4 font-medium">Estado</th>
							<th className="py-2 pr-4 font-medium">Alta</th>
						</tr>
					</thead>
					<tbody>
						{courses.map((course) => (
							<tr key={course.id} className="border-b border-ink-15/60">
								<td className="py-2.5 pr-4">
									<a href={`/admin/cursos/${course.slug}`} className="text-marian-blue hover:text-marian-blue-deep hover:underline">
										{course.title}
									</a>
								</td>
								<td className="py-2.5 pr-4 text-ink-60">{course.slug}</td>
								<td className="py-2.5 pr-4">
									{course.published ? (
										<span className="badge badge-success badge-sm">Publicado</span>
									) : (
										<span className="badge badge-ghost badge-sm">Borrador</span>
									)}
								</td>
								<td className="py-2.5 pr-4 text-ink-60">{new Date(course.createdAt).toLocaleDateString("es-AR")}</td>
							</tr>
						))}
					</tbody>
				</table>
			)}
		</div>
	);
}
