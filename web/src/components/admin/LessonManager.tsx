import { useEffect, useState, type FormEvent } from "react";
import { createLesson, updateLesson, deleteLesson, type AdminLesson } from "../../lib/api/admin";
import { ApiError } from "../../lib/api/client";
import ContentManager from "./ContentManager";

type Props = {
	slug: string;
	lessons: AdminLesson[];
	onChange: () => void;
};

/** Lecciones de un curso: listado (cada una expandible a su contenido, ver
 * ContentManager), renombrar, borrar, y alta. Reordenar lecciones no está
 * soportado todavía — ver LessonRepo.UpdateTitle en el API sobre por qué. */
export default function LessonManager({ slug, lessons, onChange }: Props) {
	const [expanded, setExpanded] = useState<number | null>(null);
	const [editingOrder, setEditingOrder] = useState<number | null>(null);
	const [editTitle, setEditTitle] = useState("");
	const [busyOrder, setBusyOrder] = useState<number | null>(null);
	const [rowError, setRowError] = useState<string | null>(null);

	const [newTitle, setNewTitle] = useState("");
	const [newOrder, setNewOrder] = useState(lessons.length + 1);
	const [creating, setCreating] = useState(false);
	const [createError, setCreateError] = useState<string | null>(null);

	// Solo sigue la CANTIDAD de lecciones, no el array entero: así, si lo
	// que cambió fue el contenido de una lección (ContentManager pidiendo un
	// refetch), este campo no te pisa lo que estabas por escribir acá.
	useEffect(() => {
		setNewOrder(lessons.length + 1);
	}, [lessons.length]);

	async function handleCreate(e: FormEvent<HTMLFormElement>) {
		e.preventDefault();
		setCreateError(null);
		setCreating(true);
		try {
			await createLesson(slug, { title: newTitle, order: newOrder });
			setNewTitle("");
			onChange();
		} catch (err) {
			setCreateError(err instanceof ApiError ? err.message : "No se pudo crear la lección.");
		} finally {
			setCreating(false);
		}
	}

	async function handleRename(order: number) {
		setRowError(null);
		setBusyOrder(order);
		try {
			await updateLesson(slug, order, editTitle);
			setEditingOrder(null);
			onChange();
		} catch (err) {
			setRowError(err instanceof ApiError ? err.message : "No se pudo renombrar.");
		} finally {
			setBusyOrder(null);
		}
	}

	async function handleDelete(order: number) {
		if (!window.confirm("¿Borrar esta lección y todo su contenido? No hay vuelta atrás.")) return;
		setRowError(null);
		setBusyOrder(order);
		try {
			await deleteLesson(slug, order);
			onChange();
		} catch (err) {
			setRowError(err instanceof ApiError ? err.message : "No se pudo borrar la lección.");
			setBusyOrder(null);
		}
	}

	return (
		<div className="flex flex-col gap-6">
			<h2 className="font-display text-xl font-semibold text-marian-blue-deep">Lecciones</h2>

			{rowError && (
				<p role="alert" className="font-ui text-sm text-sacred-red">
					{rowError}
				</p>
			)}

			{lessons.length === 0 ? (
				<p className="font-ui text-sm text-ink-60">Todavía no hay lecciones.</p>
			) : (
				<ol className="flex flex-col gap-2">
					{lessons.map((lesson) => (
						<li key={lesson.order} className="rounded-[4px] border border-ink-15">
							<div className="flex items-center gap-3 p-3">
								<button
									type="button"
									onClick={() => setExpanded(expanded === lesson.order ? null : lesson.order)}
									aria-label={expanded === lesson.order ? "Contraer" : "Expandir"}
									className="font-ui text-sm text-marian-blue hover:text-marian-blue-deep"
								>
									{expanded === lesson.order ? "▾" : "▸"}
								</button>

								{editingOrder === lesson.order ? (
									<input
										type="text"
										value={editTitle}
										onChange={(e) => setEditTitle(e.target.value)}
										className="input input-bordered input-sm flex-1"
									/>
								) : (
									<span className="flex-1 font-ui text-sm text-ink-80">
										{lesson.order}. {lesson.title}
									</span>
								)}

								{editingOrder === lesson.order ? (
									<>
										<button
											type="button"
											disabled={busyOrder === lesson.order}
											onClick={() => void handleRename(lesson.order)}
											className="btn btn-primary btn-sm"
										>
											Guardar
										</button>
										<button type="button" onClick={() => setEditingOrder(null)} className="btn btn-outline btn-sm">
											Cancelar
										</button>
									</>
								) : (
									<>
										<button
											type="button"
											onClick={() => {
												setEditingOrder(lesson.order);
												setEditTitle(lesson.title);
											}}
											className="btn btn-outline btn-sm"
										>
											Renombrar
										</button>
										<button
											type="button"
											disabled={busyOrder === lesson.order}
											onClick={() => void handleDelete(lesson.order)}
											className="btn btn-outline btn-sm border-sacred-red text-sacred-red hover:bg-sacred-red-tint"
										>
											Borrar
										</button>
									</>
								)}
							</div>

							{expanded === lesson.order && (
								<div className="border-t border-ink-15 bg-paper p-4">
									<ContentManager slug={slug} lessonOrder={lesson.order} content={lesson.content} onChange={onChange} />
								</div>
							)}
						</li>
					))}
				</ol>
			)}

			<form onSubmit={handleCreate} className="flex flex-wrap items-end gap-3 rounded-[4px] border border-ink-15 bg-paper p-4">
				<label className="flex flex-col gap-1">
					<span className="font-ui text-sm text-ink-80">Título</span>
					<input
						type="text"
						required
						value={newTitle}
						onChange={(e) => setNewTitle(e.target.value)}
						className="input input-bordered input-sm"
					/>
				</label>
				<label className="flex flex-col gap-1">
					<span className="font-ui text-sm text-ink-80">Orden</span>
					<input
						type="number"
						required
						min={1}
						value={newOrder}
						onChange={(e) => setNewOrder(Number(e.target.value))}
						className="input input-bordered input-sm w-20"
					/>
				</label>
				<button type="submit" disabled={creating} className="btn btn-cta btn-sm">
					{creating ? "Agregando…" : "Agregar lección"}
				</button>
				{createError && (
					<p role="alert" className="w-full font-ui text-sm text-sacred-red">
						{createError}
					</p>
				)}
			</form>
		</div>
	);
}
