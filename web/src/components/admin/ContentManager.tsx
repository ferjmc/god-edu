import { useState, type FormEvent } from "react";
import { createContent, uploadPDFContent, updateContent, deleteContent, type AdminLessonContent } from "../../lib/api/admin";
import { ApiError } from "../../lib/api/client";

type Props = {
	slug: string;
	lessonOrder: number;
	content: AdminLessonContent[];
	onChange: () => void;
};

const typeLabels: Record<AdminLessonContent["type"], string> = {
	video: "Video",
	markdown: "Texto",
	pdf: "PDF",
};

/** Contenido de una lección puntual: listado + edición inline + alta. El
 * tipo de una pieza ya cargada no se puede cambiar, ni el archivo de un PDF
 * — para eso hay que borrarla y cargar una nueva (mismo criterio que el
 * API, ver AdminHandler.UpdateContent). */
export default function ContentManager({ slug, lessonOrder, content, onChange }: Props) {
	const [editingOrder, setEditingOrder] = useState<number | null>(null);
	const [editTitle, setEditTitle] = useState("");
	const [editYoutubeUrl, setEditYoutubeUrl] = useState("");
	const [editBody, setEditBody] = useState("");
	const [busyOrder, setBusyOrder] = useState<number | null>(null);
	const [rowError, setRowError] = useState<string | null>(null);

	const [newType, setNewType] = useState<"video" | "markdown" | "pdf">("video");
	const [newTitle, setNewTitle] = useState("");
	const [newYoutubeUrl, setNewYoutubeUrl] = useState("");
	const [newBody, setNewBody] = useState("");
	const [newFile, setNewFile] = useState<File | null>(null);
	const [creating, setCreating] = useState(false);
	const [createError, setCreateError] = useState<string | null>(null);

	function startEdit(item: AdminLessonContent) {
		setEditingOrder(item.order);
		setEditTitle(item.title);
		setEditYoutubeUrl(item.youtubeUrl ?? "");
		setEditBody(item.body ?? "");
		setRowError(null);
	}

	async function handleSaveEdit(item: AdminLessonContent) {
		setRowError(null);
		setBusyOrder(item.order);
		try {
			await updateContent(slug, lessonOrder, item.order, {
				title: editTitle,
				youtubeUrl: item.type === "video" ? editYoutubeUrl : undefined,
				body: item.type === "markdown" ? editBody : undefined,
			});
			setEditingOrder(null);
			onChange();
		} catch (err) {
			setRowError(err instanceof ApiError ? err.message : "No se pudo guardar.");
		} finally {
			setBusyOrder(null);
		}
	}

	async function handleDelete(order: number) {
		if (!window.confirm("¿Borrar este contenido? No hay vuelta atrás.")) return;
		setRowError(null);
		setBusyOrder(order);
		try {
			await deleteContent(slug, lessonOrder, order);
			onChange();
		} catch (err) {
			setRowError(err instanceof ApiError ? err.message : "No se pudo borrar.");
			setBusyOrder(null);
		}
	}

	function resetNewForm() {
		setNewTitle("");
		setNewYoutubeUrl("");
		setNewBody("");
		setNewFile(null);
	}

	async function handleCreate(e: FormEvent<HTMLFormElement>) {
		e.preventDefault();
		setCreateError(null);
		setCreating(true);
		try {
			if (newType === "pdf") {
				if (!newFile) throw new Error("Elegí un archivo PDF.");
				await uploadPDFContent(slug, lessonOrder, newTitle, newFile);
			} else {
				await createContent(slug, lessonOrder, {
					title: newTitle,
					type: newType,
					youtubeUrl: newType === "video" ? newYoutubeUrl : undefined,
					body: newType === "markdown" ? newBody : undefined,
				});
			}
			resetNewForm();
			onChange();
		} catch (err) {
			setCreateError(err instanceof ApiError || err instanceof Error ? err.message : "No se pudo crear el contenido.");
		} finally {
			setCreating(false);
		}
	}

	return (
		<div className="flex flex-col gap-4">
			{rowError && (
				<p role="alert" className="font-ui text-sm text-sacred-red">
					{rowError}
				</p>
			)}

			{content.length === 0 ? (
				<p className="font-ui text-sm text-ink-60">Todavía no hay contenido en esta lección.</p>
			) : (
				<ul className="flex flex-col gap-2">
					{content.map((item) => (
						<li key={item.order} className="rounded-[4px] border border-ink-15 bg-base-100 p-3">
							{editingOrder === item.order ? (
								<div className="flex flex-col gap-2">
									<input
										type="text"
										value={editTitle}
										onChange={(e) => setEditTitle(e.target.value)}
										className="input input-bordered input-sm"
									/>
									{item.type === "video" && (
										<input
											type="url"
											value={editYoutubeUrl}
											onChange={(e) => setEditYoutubeUrl(e.target.value)}
											placeholder="https://youtube.com/watch?v=..."
											className="input input-bordered input-sm"
										/>
									)}
									{item.type === "markdown" && (
										<textarea
											rows={4}
											value={editBody}
											onChange={(e) => setEditBody(e.target.value)}
											className="textarea textarea-bordered textarea-sm"
										/>
									)}
									<div className="flex gap-2">
										<button
											type="button"
											disabled={busyOrder === item.order}
											onClick={() => void handleSaveEdit(item)}
											className="btn btn-primary btn-sm"
										>
											Guardar
										</button>
										<button type="button" onClick={() => setEditingOrder(null)} className="btn btn-outline btn-sm">
											Cancelar
										</button>
									</div>
								</div>
							) : (
								<div className="flex items-center gap-3">
									<span className="badge badge-ghost badge-sm">{typeLabels[item.type]}</span>
									<span className="flex-1 font-ui text-sm text-ink-80">{item.title}</span>
									<button type="button" onClick={() => startEdit(item)} className="btn btn-outline btn-sm">
										Editar
									</button>
									<button
										type="button"
										disabled={busyOrder === item.order}
										onClick={() => void handleDelete(item.order)}
										className="btn btn-outline btn-sm border-sacred-red text-sacred-red hover:bg-sacred-red-tint"
									>
										Borrar
									</button>
								</div>
							)}
						</li>
					))}
				</ul>
			)}

			<form onSubmit={handleCreate} className="flex flex-col gap-3 rounded-[4px] border border-ink-15 bg-base-100 p-4">
				<div className="flex flex-wrap items-end gap-3">
					<label className="flex flex-col gap-1">
						<span className="font-ui text-sm text-ink-80">Tipo</span>
						<select
							value={newType}
							onChange={(e) => setNewType(e.target.value as typeof newType)}
							className="select select-bordered select-sm"
						>
							<option value="video">Video</option>
							<option value="markdown">Texto</option>
							<option value="pdf">PDF</option>
						</select>
					</label>
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
				</div>

				{newType === "video" && (
					<label className="flex flex-col gap-1">
						<span className="font-ui text-sm text-ink-80">URL de YouTube</span>
						<input
							type="url"
							required
							value={newYoutubeUrl}
							onChange={(e) => setNewYoutubeUrl(e.target.value)}
							placeholder="https://youtube.com/watch?v=..."
							className="input input-bordered input-sm"
						/>
					</label>
				)}

				{newType === "markdown" && (
					<label className="flex flex-col gap-1">
						<span className="font-ui text-sm text-ink-80">Texto (markdown)</span>
						<textarea
							rows={4}
							required
							value={newBody}
							onChange={(e) => setNewBody(e.target.value)}
							className="textarea textarea-bordered textarea-sm"
						/>
					</label>
				)}

				{newType === "pdf" && (
					<label className="flex flex-col gap-1">
						<span className="font-ui text-sm text-ink-80">Archivo PDF</span>
						<input
							type="file"
							accept="application/pdf"
							required
							onChange={(e) => setNewFile(e.target.files?.[0] ?? null)}
							className="file-input file-input-bordered file-input-sm"
						/>
					</label>
				)}

				{createError && (
					<p role="alert" className="font-ui text-sm text-sacred-red">
						{createError}
					</p>
				)}

				<button type="submit" disabled={creating} className="btn btn-cta btn-sm w-fit">
					{creating ? "Agregando…" : "Agregar contenido"}
				</button>
			</form>
		</div>
	);
}
