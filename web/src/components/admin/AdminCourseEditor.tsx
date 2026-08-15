import { useEffect, useState, type FormEvent } from "react";
import { useStore } from "@nanostores/react";
import { $auth, initAuth } from "../../stores/auth";
import { getAdminCourse, updateCourseDetails, setCoursePublished, deleteCourse, type AdminCourseDetail } from "../../lib/api/admin";
import { ApiError } from "../../lib/api/client";
import LessonManager from "./LessonManager";

type Props = {
	slug: string;
};

/** Pantalla de edición completa de un curso: datos básicos, publicar/
 * despublicar, borrar, y la currícula (lecciones + contenido, delegado a
 * LessonManager). Gated por sesión + rol ADMIN, mismo patrón que
 * AdminCoursesList — la página es un shell estático, este island resuelve
 * todo client-side.
 *
 * Sin edición optimista: cada mutación vuelve a pedir el curso entero
 * (reload). Para el volumen de este panel (un admin, cargando contenido de
 * a poco) es más simple y más difícil de romper que ir parcheando el
 * estado local a mano en cada acción. */
export default function AdminCourseEditor({ slug }: Props) {
	const auth = useStore($auth);
	const [course, setCourse] = useState<AdminCourseDetail | null>(null);
	const [loadError, setLoadError] = useState<string | null>(null);

	const [title, setTitle] = useState("");
	const [description, setDescription] = useState("");
	const [savingDetails, setSavingDetails] = useState(false);
	const [detailsError, setDetailsError] = useState<string | null>(null);

	const [togglingPublished, setTogglingPublished] = useState(false);
	const [deleting, setDeleting] = useState(false);
	const [deleteError, setDeleteError] = useState<string | null>(null);

	useEffect(() => {
		if (auth.status === "loading") void initAuth();
	}, [auth.status]);

	async function reload() {
		try {
			const data = await getAdminCourse(slug);
			setCourse(data);
			setTitle(data.title);
			setDescription(data.description ?? "");
		} catch (err) {
			setLoadError(err instanceof ApiError && err.status === 404 ? "No encontramos ese curso." : "No pudimos cargar el curso.");
		}
	}

	useEffect(() => {
		if (auth.status !== "authenticated" || auth.user.role !== "ADMIN") return;
		void reload();
		// reload es estable en los hechos (no depende de nada que cambie sin
		// que slug también cambie) — no hace falta en las deps.
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [auth.status, slug]);

	async function handleSaveDetails(e: FormEvent<HTMLFormElement>) {
		e.preventDefault();
		setDetailsError(null);
		setSavingDetails(true);
		try {
			await updateCourseDetails(slug, { title, description: description.trim() === "" ? null : description });
			await reload();
		} catch (err) {
			setDetailsError(err instanceof ApiError ? err.message : "No se pudo guardar.");
		} finally {
			setSavingDetails(false);
		}
	}

	async function handleTogglePublished() {
		if (!course) return;
		setTogglingPublished(true);
		try {
			await setCoursePublished(slug, !course.published);
			await reload();
		} catch (err) {
			setDetailsError(err instanceof ApiError ? err.message : "No se pudo actualizar el estado.");
		} finally {
			setTogglingPublished(false);
		}
	}

	async function handleDelete() {
		if (!window.confirm("¿Borrar este curso? Se van a borrar también sus lecciones y contenido. No hay vuelta atrás.")) return;
		setDeleting(true);
		setDeleteError(null);
		try {
			await deleteCourse(slug);
			window.location.assign("/admin/cursos");
		} catch (err) {
			setDeleteError(err instanceof ApiError ? err.message : "No se pudo borrar el curso.");
			setDeleting(false);
		}
	}

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

	if (loadError) {
		return (
			<p role="alert" className="font-ui text-sm text-sacred-red">
				{loadError}
			</p>
		);
	}

	if (!course) {
		return <span className="loading loading-spinner loading-md text-marian-blue" aria-label="Cargando curso" />;
	}

	return (
		<div className="flex flex-col gap-10">
			<div className="flex flex-wrap items-center justify-between gap-4">
				<span className={`badge ${course.published ? "badge-success" : "badge-ghost"}`}>
					{course.published ? "Publicado" : "Borrador"}
				</span>
				<div className="flex gap-3">
					<button
						type="button"
						disabled={togglingPublished}
						onClick={() => void handleTogglePublished()}
						className="btn btn-outline btn-sm"
					>
						{togglingPublished ? "Guardando…" : course.published ? "Despublicar" : "Publicar"}
					</button>
					<button
						type="button"
						disabled={deleting}
						onClick={() => void handleDelete()}
						className="btn btn-outline btn-sm border-sacred-red text-sacred-red hover:bg-sacred-red-tint"
					>
						{deleting ? "Borrando…" : "Borrar curso"}
					</button>
				</div>
			</div>

			<form onSubmit={handleSaveDetails} className="flex max-w-xl flex-col gap-4">
				<label className="flex flex-col gap-1">
					<span className="font-ui text-sm text-ink-80">Título</span>
					<input
						type="text"
						required
						value={title}
						onChange={(e) => setTitle(e.target.value)}
						className="input input-bordered w-full"
					/>
				</label>
				<label className="flex flex-col gap-1">
					<span className="font-ui text-sm text-ink-80">Descripción</span>
					<textarea
						rows={3}
						value={description}
						onChange={(e) => setDescription(e.target.value)}
						className="textarea textarea-bordered w-full"
					/>
				</label>

				{detailsError && (
					<p role="alert" className="font-ui text-sm text-sacred-red">
						{detailsError}
					</p>
				)}
				{deleteError && (
					<p role="alert" className="font-ui text-sm text-sacred-red">
						{deleteError}
					</p>
				)}

				<button type="submit" disabled={savingDetails} className="btn btn-primary w-fit">
					{savingDetails ? "Guardando…" : "Guardar cambios"}
				</button>
			</form>

			<LessonManager slug={slug} lessons={course.lessons} onChange={() => void reload()} />
		</div>
	);
}
