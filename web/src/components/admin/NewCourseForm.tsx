import { useState, type FormEvent } from "react";
import { createCourse } from "../../lib/api/admin";
import { ApiError } from "../../lib/api/client";

/** ej. "Iniciación a la Fe" → "iniciacion-a-la-fe" — coincide con
 * slugPattern en el API (minúsculas, números, guiones). */
function slugify(title: string): string {
	return title
		.toLowerCase()
		.normalize("NFD")
		.replace(/[\u0300-\u036f]/g, "")
		.replace(/[^a-z0-9]+/g, "-")
		.replace(/^-+|-+$/g, "");
}

type Props = {
	onCreated: (slug: string) => void;
};

/** Formulario mínimo para arrancar un curso nuevo: título + slug
 * (autogenerado del título, pero editable). El resto — descripción,
 * lecciones, contenido — se carga después en la pantalla de edición del
 * curso ya creado (ver AdminCourseEditor), no acá. */
export default function NewCourseForm({ onCreated }: Props) {
	const [open, setOpen] = useState(false);
	const [title, setTitle] = useState("");
	const [slug, setSlug] = useState("");
	const [slugTouched, setSlugTouched] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [submitting, setSubmitting] = useState(false);

	function handleTitleChange(value: string) {
		setTitle(value);
		if (!slugTouched) setSlug(slugify(value));
	}

	async function handleSubmit(e: FormEvent<HTMLFormElement>) {
		e.preventDefault();
		setError(null);
		setSubmitting(true);
		try {
			const course = await createCourse({ title, slug });
			onCreated(course.slug);
		} catch (err) {
			setError(err instanceof ApiError ? err.message : "No se pudo crear el curso.");
			setSubmitting(false);
		}
	}

	if (!open) {
		return (
			<button type="button" className="btn btn-cta" onClick={() => setOpen(true)}>
				Nuevo curso
			</button>
		);
	}

	return (
		<form onSubmit={handleSubmit} className="flex max-w-md flex-col gap-4 rounded-[4px] border border-ink-15 bg-paper p-6">
			<label className="flex flex-col gap-1">
				<span className="font-ui text-sm text-ink-80">Título</span>
				<input
					type="text"
					required
					value={title}
					onChange={(e) => handleTitleChange(e.target.value)}
					className="input input-bordered w-full"
				/>
			</label>
			<label className="flex flex-col gap-1">
				<span className="font-ui text-sm text-ink-80">Slug (URL)</span>
				<input
					type="text"
					required
					pattern="[a-z0-9]+(-[a-z0-9]+)*"
					value={slug}
					onChange={(e) => {
						setSlugTouched(true);
						setSlug(e.target.value);
					}}
					className="input input-bordered w-full"
				/>
			</label>

			{error && (
				<p role="alert" className="font-ui text-sm text-sacred-red">
					{error}
				</p>
			)}

			<div className="flex gap-3">
				<button type="submit" disabled={submitting} className="btn btn-cta">
					{submitting ? "Creando…" : "Crear curso"}
				</button>
				<button type="button" className="btn btn-outline" onClick={() => setOpen(false)}>
					Cancelar
				</button>
			</div>
		</form>
	);
}
