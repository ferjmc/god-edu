import { useEffect, useRef, useState } from "react";
import { listUserLessonActivity, type AdminUser, type AdminUserLessonActivity } from "../../lib/api/admin";
import { ApiError } from "../../lib/api/client";

type Props = {
	user: AdminUser;
	roleLabel: string;
	/** Hace el PATCH y refresca la lista. Si falla, tira el error para que
	 * el modal lo muestre junto al toggle. */
	onVerifiedChange: (verified: boolean) => Promise<void>;
	onClose: () => void;
};

type CourseGroup = {
	courseSlug: string;
	courseTitle: string;
	lessons: AdminUserLessonActivity[];
};

function formatDateTime(value: string | null): string {
	if (!value) return "—";
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) return "—";
	return date.toLocaleString("es-AR", { dateStyle: "short", timeStyle: "short" });
}

/** La API ya devuelve la actividad ordenada por curso, así que alcanza con
 * cortar donde cambia el slug. */
function groupByCourse(activity: AdminUserLessonActivity[]): CourseGroup[] {
	const groups: CourseGroup[] = [];
	for (const item of activity) {
		const last = groups[groups.length - 1];
		if (last && last.courseSlug === item.courseSlug) {
			last.lessons.push(item);
		} else {
			groups.push({ courseSlug: item.courseSlug, courseTitle: item.courseTitle, lessons: [item] });
		}
	}
	return groups;
}

/** Detalle de un usuario en un <dialog> de daisyUI: datos de la cuenta,
 * toggle de verificación y las lecciones que abrió/completó. Se monta solo
 * mientras está abierto, así cada apertura vuelve a pedir la actividad. */
export default function AdminUserDetailModal({ user, roleLabel, onVerifiedChange, onClose }: Props) {
	const dialogRef = useRef<HTMLDialogElement>(null);
	const [activity, setActivity] = useState<AdminUserLessonActivity[] | null>(null);
	const [activityError, setActivityError] = useState<string | null>(null);
	const [savingVerified, setSavingVerified] = useState(false);
	const [verifiedError, setVerifiedError] = useState<string | null>(null);

	useEffect(() => {
		dialogRef.current?.showModal();
	}, []);

	useEffect(() => {
		let ignore = false;
		setActivity(null);
		setActivityError(null);
		listUserLessonActivity(user.id)
			.then((data) => {
				if (!ignore) setActivity(data);
			})
			.catch((err) => {
				if (!ignore) setActivityError(err instanceof ApiError ? err.message : "No pudimos cargar las sesiones.");
			});
		return () => {
			ignore = true;
		};
	}, [user.id]);

	async function handleToggleVerified() {
		setVerifiedError(null);
		setSavingVerified(true);
		try {
			await onVerifiedChange(!user.emailVerified);
		} catch (err) {
			setVerifiedError(err instanceof ApiError ? err.message : "No se pudo actualizar la verificación.");
		} finally {
			setSavingVerified(false);
		}
	}

	const groups = activity ? groupByCourse(activity) : [];

	return (
		<dialog ref={dialogRef} className="modal modal-bottom sm:modal-middle" onClose={onClose} aria-labelledby="admin-user-detail-title">
			<div className="modal-box max-w-2xl bg-paper">
				<form method="dialog">
					<button className="btn btn-sm btn-circle btn-ghost absolute right-3 top-3" aria-label="Cerrar">
						✕
					</button>
				</form>

				<h2 id="admin-user-detail-title" className="pr-8 font-display text-2xl text-marian-blue-deep">
					{user.name}
				</h2>
				<p className="break-all font-ui text-sm text-ink-60">{user.email}</p>

				<dl className="mt-6 grid grid-cols-2 gap-x-6 gap-y-3 font-ui text-sm">
					<div>
						<dt className="text-ink-60">Id</dt>
						<dd>{user.id}</dd>
					</div>
					<div>
						<dt className="text-ink-60">Rol</dt>
						<dd>{roleLabel}</dd>
					</div>
					<div>
						<dt className="text-ink-60">Servicio de acceso</dt>
						<dd>{user.authProvider}</dd>
					</div>
					<div>
						<dt className="text-ink-60">Fecha de alta</dt>
						<dd>{formatDateTime(user.createdAt)}</dd>
					</div>
				</dl>

				<div className="mt-6 flex flex-col gap-2 rounded-[4px] border border-ink-15 p-4">
					<label className="flex cursor-pointer items-center justify-between gap-4 font-ui text-sm">
						<span>
							<span className="font-medium text-ink">Cuenta verificada</span>
							<span className="block text-ink-60">
								{user.emailVerified ? "El email de esta cuenta está verificado." : "Esta cuenta todavía no verificó su email."}
							</span>
						</span>
						<input
							type="checkbox"
							className="toggle toggle-success"
							checked={user.emailVerified}
							disabled={savingVerified}
							onChange={() => void handleToggleVerified()}
						/>
					</label>
					{verifiedError && (
						<p role="alert" className="font-ui text-sm text-sacred-red">
							{verifiedError}
						</p>
					)}
				</div>

				<h3 className="mt-8 font-display text-xl text-marian-blue-deep">Sesiones vistas</h3>
				<div className="mt-3">
					{activityError ? (
						<p role="alert" className="font-ui text-sm text-sacred-red">
							{activityError}
						</p>
					) : !activity ? (
						<span className="loading loading-spinner loading-md text-marian-blue" aria-label="Cargando sesiones" />
					) : groups.length === 0 ? (
						<p className="font-ui text-sm text-ink-60">Este usuario todavía no abrió ninguna sesión.</p>
					) : (
						<div className="flex flex-col gap-5">
							{groups.map((group) => (
								<section key={group.courseSlug}>
									<h4 className="font-ui text-sm font-medium text-ink">{group.courseTitle}</h4>
									<ul className="mt-2 flex flex-col divide-y divide-ink-15/60 rounded-[4px] border border-ink-15">
										{group.lessons.map((lesson) => (
											<li key={lesson.lessonOrder} className="flex flex-wrap items-center gap-x-3 gap-y-1 px-3 py-2 font-ui text-sm">
												<span className="text-ink-60">{lesson.lessonOrder}.</span>
												<span className="flex-1">{lesson.lessonTitle}</span>
												{lesson.completed ? (
													<span className="badge badge-success badge-sm">Completada</span>
												) : (
													<span className="badge badge-ghost badge-sm">Vista</span>
												)}
												<span className="w-full text-xs text-ink-60 sm:w-auto">
													{lesson.completed
														? `Completada ${formatDateTime(lesson.completedAt)}`
														: `Última vista ${formatDateTime(lesson.lastViewedAt)}`}
												</span>
											</li>
										))}
									</ul>
								</section>
							))}
						</div>
					)}
				</div>
			</div>
			<form method="dialog" className="modal-backdrop">
				<button aria-label="Cerrar">cerrar</button>
			</form>
		</dialog>
	);
}
