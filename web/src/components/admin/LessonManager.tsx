import { useEffect, useState, type FormEvent } from "react";
import { createLesson, updateLesson, deleteLesson, type AdminClass, type AdminLesson } from "../../lib/api/admin";
import { ApiError } from "../../lib/api/client";
import ClassManager from "./ClassManager";
import ContentManager from "./ContentManager";

type Props = {
	slug: string;
	classes: AdminClass[];
	lessons: AdminLesson[];
	onChange: () => void;
};

/** Currícula de un curso: clases (ver ClassManager) y sesiones — en el API
 * y la base siguen siendo "lessons". Sesiones: listado (cada una expandible
 * a su contenido, ver ContentManager), editar título y clase, borrar, y
 * alta. Reordenar sesiones no está soportado todavía — ver
 * LessonRepo.UpdateTitle en el API sobre por qué. */
export default function LessonManager({ slug, classes, lessons, onChange }: Props) {
	const [expanded, setExpanded] = useState<number | null>(null);
	const [editingOrder, setEditingOrder] = useState<number | null>(null);
	const [editTitle, setEditTitle] = useState("");
	const [editClassOrder, setEditClassOrder] = useState<number | null>(null);
	const [busyOrder, setBusyOrder] = useState<number | null>(null);
	const [rowError, setRowError] = useState<string | null>(null);

	const [newTitle, setNewTitle] = useState("");
	const [newOrder, setNewOrder] = useState(lessons.length + 1);
	const [newClassOrder, setNewClassOrder] = useState<number | null>(null);
	const [creating, setCreating] = useState(false);
	const [createError, setCreateError] = useState<string | null>(null);

	// Solo sigue la CANTIDAD de sesiones, no el array entero: así, si lo
	// que cambió fue el contenido de una sesión (ContentManager pidiendo un
	// refetch), este campo no te pisa lo que estabas por escribir acá.
	useEffect(() => {
		setNewOrder(lessons.length + 1);
	}, [lessons.length]);

	// Si se borró la clase elegida para la próxima sesión, volver a "sin clase".
	useEffect(() => {
		if (newClassOrder !== null && !classes.some((c) => c.order === newClassOrder)) setNewClassOrder(null);
	}, [classes, newClassOrder]);

	const classLabel = (order: number | undefined) => {
		const c = classes.find((c) => c.order === order);
		if (!c) return "Sin clase";
		return c.published ? c.title : `${c.title} (oculta)`;
	};

	async function handleCreate(e: FormEvent<HTMLFormElement>) {
		e.preventDefault();
		setCreateError(null);
		setCreating(true);
		try {
			await createLesson(slug, { title: newTitle, order: newOrder, classOrder: newClassOrder });
			setNewTitle("");
			onChange();
		} catch (err) {
			setCreateError(err instanceof ApiError ? err.message : "No se pudo crear la sesión.");
		} finally {
			setCreating(false);
		}
	}

	async function handleSave(order: number) {
		setRowError(null);
		setBusyOrder(order);
		try {
			await updateLesson(slug, order, { title: editTitle, classOrder: editClassOrder });
			setEditingOrder(null);
			onChange();
		} catch (err) {
			setRowError(err instanceof ApiError ? err.message : "No se pudo guardar la sesión.");
		} finally {
			setBusyOrder(null);
		}
	}

	async function handleDelete(order: number) {
		if (!window.confirm("¿Borrar esta sesión y todo su contenido? No hay vuelta atrás.")) return;
		setRowError(null);
		setBusyOrder(order);
		try {
			await deleteLesson(slug, order);
			onChange();
		} catch (err) {
			setRowError(err instanceof ApiError ? err.message : "No se pudo borrar la sesión.");
			setBusyOrder(null);
		}
	}

	return (
		<div className="flex flex-col gap-10">
			<ClassManager slug={slug} classes={classes} onChange={onChange} />

			<div className="flex flex-col gap-6">
				<h2 className="font-display text-xl font-semibold text-marian-blue-deep">Sesiones</h2>

				{rowError && (
					<p role="alert" className="font-ui text-sm text-sacred-red">
						{rowError}
					</p>
				)}

				{lessons.length === 0 ? (
					<p className="font-ui text-sm text-ink-60">Todavía no hay sesiones.</p>
				) : (
					<ol className="flex flex-col gap-2">
						{lessons.map((lesson) => (
							<li key={lesson.order} className="rounded-[4px] border border-ink-15">
								<div className="flex flex-wrap items-center gap-3 p-3">
									<button
										type="button"
										onClick={() => setExpanded(expanded === lesson.order ? null : lesson.order)}
										aria-label={expanded === lesson.order ? "Contraer" : "Expandir"}
										className="font-ui text-sm text-marian-blue hover:text-marian-blue-deep"
									>
										{expanded === lesson.order ? "▾" : "▸"}
									</button>

									{editingOrder === lesson.order ? (
										<>
											<input
												type="text"
												value={editTitle}
												onChange={(e) => setEditTitle(e.target.value)}
												className="input input-bordered input-sm flex-1"
											/>
											<ClassSelect classes={classes} value={editClassOrder} onChange={setEditClassOrder} />
											<button
												type="button"
												disabled={busyOrder === lesson.order}
												onClick={() => void handleSave(lesson.order)}
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
											<span className="flex-1 font-ui text-sm text-ink-80">
												{lesson.order}. {lesson.title}
											</span>
											<span className="font-ui text-xs text-ink-60">{classLabel(lesson.classOrder)}</span>
											<button
												type="button"
												onClick={() => {
													setEditingOrder(lesson.order);
													setEditTitle(lesson.title);
													setEditClassOrder(lesson.classOrder ?? null);
												}}
												className="btn btn-outline btn-sm"
											>
												Editar
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
					<label className="flex flex-col gap-1">
						<span className="font-ui text-sm text-ink-80">Clase</span>
						<ClassSelect classes={classes} value={newClassOrder} onChange={setNewClassOrder} />
					</label>
					<button type="submit" disabled={creating} className="btn btn-cta btn-sm">
						{creating ? "Agregando…" : "Agregar sesión"}
					</button>
					{createError && (
						<p role="alert" className="w-full font-ui text-sm text-sacred-red">
							{createError}
						</p>
					)}
				</form>
			</div>
		</div>
	);
}

function ClassSelect({
	classes,
	value,
	onChange,
}: {
	classes: AdminClass[];
	value: number | null;
	onChange: (classOrder: number | null) => void;
}) {
	return (
		<select
			value={value ?? ""}
			onChange={(e) => onChange(e.target.value === "" ? null : Number(e.target.value))}
			className="select select-bordered select-sm"
			aria-label="Clase"
		>
			<option value="">Sin clase</option>
			{classes.map((c) => (
				<option key={c.order} value={c.order}>
					{c.order}. {c.title}
					{c.published ? "" : " (oculta)"}
				</option>
			))}
		</select>
	);
}
