import { useEffect, useState, type FormEvent } from "react";
import { createClass, updateClass, deleteClass, setClassPublished, type AdminClass } from "../../lib/api/admin";
import { ApiError } from "../../lib/api/client";

type Props = {
	slug: string;
	classes: AdminClass[];
	onChange: () => void;
};

/** Clases de un curso: listado, renombrar, borrar y alta. Una clase solo
 * agrupa sesiones — asignar una sesión a una clase se hace desde la sesión
 * (ver LessonManager). Mismo patrón que LessonManager: cada mutación
 * dispara onChange y el editor vuelve a pedir el curso entero. */
export default function ClassManager({ slug, classes, onChange }: Props) {
	const [editingOrder, setEditingOrder] = useState<number | null>(null);
	const [editTitle, setEditTitle] = useState("");
	const [busyOrder, setBusyOrder] = useState<number | null>(null);
	const [rowError, setRowError] = useState<string | null>(null);

	const nextOrder = (classes.at(-1)?.order ?? 0) + 1;
	const [newTitle, setNewTitle] = useState("");
	const [newOrder, setNewOrder] = useState(nextOrder);
	const [creating, setCreating] = useState(false);
	const [createError, setCreateError] = useState<string | null>(null);

	useEffect(() => {
		setNewOrder(nextOrder);
	}, [nextOrder]);

	async function handleCreate(e: FormEvent<HTMLFormElement>) {
		e.preventDefault();
		setCreateError(null);
		setCreating(true);
		try {
			await createClass(slug, { title: newTitle, order: newOrder });
			setNewTitle("");
			onChange();
		} catch (err) {
			setCreateError(err instanceof ApiError ? err.message : "No se pudo crear la clase.");
		} finally {
			setCreating(false);
		}
	}

	async function handleRename(order: number) {
		setRowError(null);
		setBusyOrder(order);
		try {
			await updateClass(slug, order, editTitle);
			setEditingOrder(null);
			onChange();
		} catch (err) {
			setRowError(err instanceof ApiError ? err.message : "No se pudo renombrar la clase.");
		} finally {
			setBusyOrder(null);
		}
	}

	async function handleTogglePublished(c: AdminClass) {
		setRowError(null);
		setBusyOrder(c.order);
		try {
			await setClassPublished(slug, c.order, !c.published);
			onChange();
		} catch (err) {
			setRowError(err instanceof ApiError ? err.message : "No se pudo cambiar la publicación de la clase.");
		} finally {
			setBusyOrder(null);
		}
	}

	async function handleDelete(order: number) {
		if (!window.confirm("¿Borrar esta clase? Sus sesiones no se borran: quedan sin clase.")) return;
		setRowError(null);
		setBusyOrder(order);
		try {
			await deleteClass(slug, order);
			onChange();
		} catch (err) {
			setRowError(err instanceof ApiError ? err.message : "No se pudo borrar la clase.");
		} finally {
			setBusyOrder(null);
		}
	}

	return (
		<div className="flex flex-col gap-4">
			<h2 className="font-display text-xl font-semibold text-marian-blue-deep">Clases</h2>

			{rowError && (
				<p role="alert" className="font-ui text-sm text-sacred-red">
					{rowError}
				</p>
			)}

			{classes.length === 0 ? (
				<p className="font-ui text-sm text-ink-60">
					Todavía no hay clases. Las sesiones se muestran como una lista simple hasta que crees la primera.
				</p>
			) : (
				<ol className="flex flex-col gap-2">
					{classes.map((c) => (
						<li key={c.order} className="flex items-center gap-3 rounded-[4px] border border-ink-15 p-3">
							{editingOrder === c.order ? (
								<>
									<input
										type="text"
										value={editTitle}
										onChange={(e) => setEditTitle(e.target.value)}
										className="input input-bordered input-sm flex-1"
									/>
									<button
										type="button"
										disabled={busyOrder === c.order}
										onClick={() => void handleRename(c.order)}
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
										{c.order}. {c.title}
									</span>
									<span className={`badge badge-sm ${c.published ? "badge-success" : "badge-ghost"}`}>
										{c.published ? "Publicada" : "Oculta"}
									</span>
									<button
										type="button"
										disabled={busyOrder === c.order}
										onClick={() => void handleTogglePublished(c)}
										className="btn btn-outline btn-sm"
									>
										{c.published ? "Ocultar" : "Publicar"}
									</button>
									<button
										type="button"
										onClick={() => {
											setEditingOrder(c.order);
											setEditTitle(c.title);
										}}
										className="btn btn-outline btn-sm"
									>
										Renombrar
									</button>
									<button
										type="button"
										disabled={busyOrder === c.order}
										onClick={() => void handleDelete(c.order)}
										className="btn btn-outline btn-sm border-sacred-red text-sacred-red hover:bg-sacred-red-tint"
									>
										Borrar
									</button>
								</>
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
					{creating ? "Agregando…" : "Agregar clase"}
				</button>
				{createError && (
					<p role="alert" className="w-full font-ui text-sm text-sacred-red">
						{createError}
					</p>
				)}
				<p className="w-full font-ui text-xs text-ink-60">
					Las clases nuevas se crean ocultas: sus sesiones no las ven los alumnos hasta que la publiques.
				</p>
			</form>
		</div>
	);
}
