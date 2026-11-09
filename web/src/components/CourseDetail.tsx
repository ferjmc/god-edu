import { useEffect, useState } from "react";
import { useStore } from "@nanostores/react";
import { $auth, initAuth } from "../stores/auth";
import { getCourse, enrollInCourse, type Course } from "../lib/api/courses";
import { groupByClass, listLessons, type LessonSummary } from "../lib/api/lessons";
import { ApiError } from "../lib/api/client";
import { CheckCircleIcon } from "./icons";

type Props = {
	slug: string;
};

/** Gates course content behind a session. The page is prerendered
 * statically (getStaticPaths in cursos/[slug].astro), so this is the only
 * place that knows whether the visitor is actually logged in — the HTML
 * itself never contains the course content for anonymous visitors.
 *
 * Un curso visible para el rol del usuario no se revela entero hasta que se
 * inscribe explícitamente (ver Enroll en el API): mientras `enrolled` es
 * false solo se muestra el resumen + botón "Inscribirme" — la lista de
 * lecciones ni se pide, para no gastar una llamada de más en algo que
 * todavía no se va a mostrar. */
export default function CourseDetail({ slug }: Props) {
	const auth = useStore($auth);
	const [course, setCourse] = useState<Course | null>(null);
	const [lessons, setLessons] = useState<LessonSummary[] | null>(null);
	const [error, setError] = useState<string | null>(null);
	// Separado de `error` a propósito: una falla acá pasa DESPUÉS de que el
	// curso (y, en handleEnroll, la inscripción) ya se cargaron bien — no
	// corresponde tapar toda la vista con un error genérico ni, peor, con un
	// mensaje de "no se pudo inscribir" cuando la inscripción sí funcionó.
	const [lessonsError, setLessonsError] = useState<string | null>(null);
	const [enrolling, setEnrolling] = useState(false);

	useEffect(() => {
		if (auth.status === "loading") void initAuth();
	}, [auth.status]);

	useEffect(() => {
		if (auth.status !== "authenticated") return;
		getCourse(slug)
			.then((courseData) => {
				setCourse(courseData);
				if (!courseData.enrolled) return;
				return listLessons(slug)
					.then(setLessons)
					.catch((err) => setLessonsError(err instanceof ApiError ? err.message : "No pudimos cargar las sesiones."));
			})
			.catch((err) => setError(err instanceof ApiError ? err.message : "No pudimos cargar el curso."));
	}, [auth.status, slug]);

	async function handleEnroll() {
		setEnrolling(true);
		try {
			const updated = await enrollInCourse(slug);
			setCourse(updated);
			try {
				setLessons(await listLessons(slug));
			} catch (err) {
				// La inscripción ya se completó (setCourse de arriba lo refleja) —
				// esto es solo que la lista de sesiones no cargó, no un fallo de
				// la inscripción en sí.
				setLessonsError(err instanceof ApiError ? err.message : "Te inscribiste, pero no pudimos cargar las sesiones.");
			}
		} catch (err) {
			setError(err instanceof ApiError ? err.message : "No se pudo completar la inscripción.");
		} finally {
			setEnrolling(false);
		}
	}

	if (auth.status === "loading") {
		return <span className="loading loading-spinner loading-md text-marian-blue" aria-label="Cargando" />;
	}

	if (auth.status === "anonymous") {
		return (
			<div className="flex flex-col items-start gap-4 rounded-[4px] border border-ink-15 bg-paper p-8">
				<p className="font-serif text-ink-80">
					Iniciá sesión o creá una cuenta para ver el contenido de este curso.
				</p>
				<div className="flex gap-3">
					<a href="/ingresar" className="btn btn-outline">
						Ingresar
					</a>
					<a href="/registro" className="btn btn-cta">
						Registrarse
					</a>
				</div>
			</div>
		);
	}

	if (error && !course) {
		return (
			<p role="alert" className="font-ui text-sm text-sacred-red">
				{error}
			</p>
		);
	}

	if (!course) {
		return <span className="loading loading-spinner loading-md text-marian-blue" aria-label="Cargando curso" />;
	}

	// Próxima lección sin completar — se resalta en la lista para que el
	// alumno no tenga que escanear los checks a mano para saber por dónde
	// seguir (mismo dato que ya usa CourseProgressCard para el CTA).
	const nextPendingOrder = lessons?.find((l) => !l.completed)?.order;
	const completedCount = lessons?.filter((l) => l.completed).length ?? 0;
	const groups = lessons ? groupByClass(lessons) : [];

	return (
		<article>
			<span className="font-ui text-xs font-semibold uppercase tracking-[0.14em] text-marian-blue">Curso</span>
			<span className="mt-3 block h-0.5 w-8 bg-liturgical-gold" aria-hidden="true" />
			<h1 className="mt-3 font-display text-3xl font-semibold text-ink sm:text-4xl">{course.title}</h1>
			{course.description && <p className="mt-4 max-w-2xl font-serif text-ink-80">{course.description}</p>}

			{(course.certificateEnabled || (course.enrolled && lessons)) && (
				<div className="mt-6 flex flex-wrap items-center gap-3">
					{course.enrolled && lessons && (
						<span className="font-ui text-xs text-ink-60">
							{completedCount}/{lessons.length} sesiones completadas
						</span>
					)}
					{course.certificateEnabled && (
						<span className="inline-flex items-center gap-1.5 rounded-[4px] border border-liturgical-gold/40 bg-liturgical-gold-tint px-2.5 py-1 font-ui text-xs font-semibold text-ink-80">
							Otorga certificado
						</span>
					)}
				</div>
			)}

			{!course.enrolled ? (
				<div className="mt-8">
					<button type="button" disabled={enrolling} onClick={() => void handleEnroll()} className="btn btn-cta">
						{enrolling ? "Inscribiendo…" : "Inscribirme"}
					</button>
					{error && (
						<p role="alert" className="mt-2 font-ui text-sm text-sacred-red">
							{error}
						</p>
					)}
				</div>
			) : lessonsError ? (
				<p role="alert" className="mt-10 font-ui text-sm text-sacred-red">
					{lessonsError}
				</p>
			) : (
				lessons &&
				lessons.length > 0 && (
					<div className="mt-10">
						<h2 className="font-display text-xl font-semibold text-marian-blue-deep">
							{groups.some((g) => g.classOrder !== null) ? "Clases" : "Sesiones"}
						</h2>
						<div className="mt-4 flex flex-col gap-6">
							{groups.map((group) => (
								<section key={group.classOrder ?? "sin-clase"}>
									{(group.classOrder !== null || groups.length > 1) && (
										<div className="mb-2 flex items-baseline justify-between gap-3 border-b border-ink-15 pb-1.5">
											<h3 className="font-display text-lg font-semibold text-ink">{group.title ?? "Otras sesiones"}</h3>
											<span className="shrink-0 font-ui text-xs text-ink-60">
												{group.lessons.filter((l) => l.completed).length}/{group.lessons.length} sesiones
											</span>
										</div>
									)}
									<ol className="flex flex-col gap-1">
										{group.lessons.map((lesson) => {
											const isNext = lesson.order === nextPendingOrder;
											return (
												<li key={lesson.order}>
													<a
														href={`/cursos/${course.slug}/${lesson.order}`}
														className={`flex items-center gap-3 rounded-[4px] border px-3 py-2.5 transition-colors hover:bg-paper-warm ${
															isNext ? "border-liturgical-gold bg-paper-warm" : "border-transparent"
														}`}
													>
														{lesson.completed ? (
															<CheckCircleIcon className="h-5 w-5 shrink-0 text-liturgical-gold" />
														) : (
															<span className="h-5 w-5 shrink-0 rounded-full border border-current" aria-hidden="true" />
														)}
														<span className="flex-1 font-ui text-sm text-ink-80">
															Sesión {lesson.order}. {lesson.title}
														</span>
														{isNext && (
															<span className="font-ui text-xs font-semibold text-marian-blue">Continuar acá →</span>
														)}
													</a>
												</li>
											);
										})}
									</ol>
								</section>
							))}
						</div>
					</div>
				)
			)}
		</article>
	);
}
