import { useEffect, useState } from "react";
import { useStore } from "@nanostores/react";
import { marked } from "marked";
import { $auth, initAuth } from "../stores/auth";
import {
	getLesson,
	listLessons,
	markLessonComplete,
	type LessonDetail as LessonDetailData,
	type LessonSummary,
} from "../lib/api/lessons";
import { ApiError } from "../lib/api/client";
import { toYoutubeEmbedUrl } from "../lib/youtube";
import { CheckCircleIcon, DownloadIcon, FileIcon } from "./icons";

type Props = {
	courseSlug: string;
	lessonNumber: number;
};

/** Gates lesson content behind a session — mismo patrón que CourseDetail.
 * No hay bloqueo secuencial: cualquier lección de un curso visible se
 * puede pedir directamente (ver LessonHandler en /api). */
export default function LessonDetail({ courseSlug, lessonNumber }: Props) {
	const auth = useStore($auth);
	const [lesson, setLesson] = useState<LessonDetailData | null>(null);
	const [curriculum, setCurriculum] = useState<LessonSummary[] | null>(null);
	const [error, setError] = useState<string | null>(null);
	const [marking, setMarking] = useState(false);
	const [markError, setMarkError] = useState<string | null>(null);

	useEffect(() => {
		if (auth.status === "loading") void initAuth();
	}, [auth.status]);

	useEffect(() => {
		if (auth.status !== "authenticated") return;
		setLesson(null);
		setMarkError(null);
		Promise.all([getLesson(courseSlug, lessonNumber), listLessons(courseSlug)])
			.then(([lessonData, curriculumData]) => {
				setLesson(lessonData);
				setCurriculum(curriculumData);
			})
			.catch((err) =>
				setError(err instanceof ApiError && err.status === 404 ? "No encontramos esa lección." : "No pudimos cargar la lección."),
			);
	}, [auth.status, courseSlug, lessonNumber]);

	/** Botón manual, no automático al entrar a la lección — ver nota en
	 * LessonHandler.MarkComplete (/api) sobre por qué. Actualiza el estado
	 * local en el momento (lección + item correspondiente en la currícula
	 * del sidebar) en vez de volver a pedir todo de nuevo al servidor. */
	async function handleMarkComplete() {
		setMarking(true);
		setMarkError(null);
		try {
			await markLessonComplete(courseSlug, lessonNumber);
			setLesson((prev) => (prev ? { ...prev, completed: true } : prev));
			setCurriculum((prev) => prev?.map((item) => (item.order === lessonNumber ? { ...item, completed: true } : item)) ?? prev);
		} catch {
			setMarkError("No pudimos registrar el progreso. Probá de nuevo.");
		} finally {
			setMarking(false);
		}
	}

	if (auth.status === "loading") {
		return <span className="loading loading-spinner loading-md text-marian-blue" aria-label="Cargando" />;
	}

	if (auth.status === "anonymous") {
		return (
			<div className="flex flex-col items-start gap-4 rounded-[4px] border border-ink-15 bg-paper p-8">
				<p className="font-serif text-ink-80">Iniciá sesión o creá una cuenta para ver esta lección.</p>
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

	if (error) {
		return (
			<p role="alert" className="font-ui text-sm text-sacred-red">
				{error}
			</p>
		);
	}

	if (!lesson || !curriculum) {
		return <span className="loading loading-spinner loading-md text-marian-blue" aria-label="Cargando lección" />;
	}

	const video = lesson.content.find((item) => item.type === "video");
	const pdfs = lesson.content.filter((item) => item.type === "pdf");
	const markdownItems = lesson.content.filter((item) => item.type === "markdown");
	const embedUrl = video?.youtubeUrl ? toYoutubeEmbedUrl(video.youtubeUrl) : null;
	const nextLesson = curriculum.find((item) => item.order === lessonNumber + 1);

	return (
		<div className="grid grid-cols-1 gap-10 lg:grid-cols-[1fr_320px] lg:items-start">
			{/* Contenido de la lección */}
			<article className="flex flex-col gap-8">
				{video && (
					<div className="aspect-video overflow-hidden rounded-[4px] border border-ink-15 bg-marian-blue-deep">
						{embedUrl ? (
							<iframe
								src={embedUrl}
								title={video.title}
								className="h-full w-full"
								allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
								allowFullScreen
							/>
						) : (
							<div className="flex h-full items-center justify-center px-6 text-center font-ui text-sm text-mercy-white/70">
								No pudimos leer la URL del video de esta lección.
							</div>
						)}
					</div>
				)}

				<div className="flex flex-wrap items-start justify-between gap-4">
					<div>
						<span className="font-ui text-xs font-semibold tracking-[0.14em] text-marian-blue uppercase">
							Lección {lesson.order}
						</span>
						<h1 className="mt-2 font-display text-3xl text-marian-blue-deep sm:text-4xl">{lesson.title}</h1>
					</div>

					{lesson.completed ? (
						<span className="flex shrink-0 items-center gap-2 rounded-[4px] bg-liturgical-gold-tint px-4 py-2 font-ui text-sm font-semibold text-marian-blue">
							<CheckCircleIcon className="h-4.5 w-4.5 text-liturgical-gold" />
							Completada
						</span>
					) : (
						<button type="button" onClick={handleMarkComplete} disabled={marking} className="btn btn-cta shrink-0">
							{marking ? "Guardando…" : "Marcar como completada"}
						</button>
					)}
				</div>

				{markError && (
					<p role="alert" className="font-ui text-sm text-sacred-red">
						{markError}
					</p>
				)}

				{markdownItems.length > 0 && (
					<div className="lesson-markdown flex flex-col gap-4">
						{markdownItems.map((item) => (
							// eslint-disable-next-line react/no-danger -- contenido cargado por el equipo de la
							// comunidad (no envío abierto de usuarios), ver nota en global.css sobre lesson-markdown.
							<div key={item.title} dangerouslySetInnerHTML={{ __html: marked.parse(item.body ?? "", { async: false }) }} />
						))}
					</div>
				)}

				{pdfs.length > 0 && (
					<div>
						<h2 className="font-display text-xl font-semibold text-marian-blue-deep">Recursos descargables</h2>
						<div className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
							{pdfs.map((resource) => (
								<a
									key={resource.title}
									href={resource.pdfUrl}
									target="_blank"
									rel="noreferrer"
									className="flex items-center gap-3 rounded-[4px] border border-ink-15 bg-paper p-4 transition-colors hover:bg-paper-warm"
								>
									<span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-[4px] bg-liturgical-gold-tint text-marian-blue">
										<FileIcon className="h-4 w-4" />
									</span>
									<span className="min-w-0 flex-1 truncate font-ui text-sm font-semibold text-ink">{resource.title}</span>
									<DownloadIcon className="h-4 w-4 shrink-0 text-marian-blue" />
								</a>
							))}
						</div>
					</div>
				)}

				{lesson.content.length === 0 && (
					<p className="font-serif text-ink-60">Todavía no hay contenido cargado para esta lección.</p>
				)}
			</article>

			{/* Currícula del curso */}
			<aside className="rounded-[4px] border border-ink-15 bg-paper p-6 shadow-gold-edge lg:sticky lg:top-20">
				<h2 className="font-display text-lg font-semibold text-marian-blue-deep">Currícula del curso</h2>

				<ol className="mt-5 flex flex-col gap-1">
					{curriculum.map((item) => {
						const isViewing = item.order === lessonNumber;
						return (
							<li key={item.order}>
								<a
									href={`/cursos/${courseSlug}/${item.order}`}
									className={`flex items-center gap-3 rounded-[4px] px-3 py-2.5 transition-colors ${
										isViewing ? "bg-marian-blue-deep text-mercy-white" : "text-ink-80 hover:bg-paper-warm"
									}`}
								>
									{item.completed ? (
										<CheckCircleIcon className="h-4.5 w-4.5 shrink-0 text-liturgical-gold" />
									) : (
										<span
											className={`h-4.5 w-4.5 shrink-0 rounded-full border ${isViewing ? "border-mercy-white" : "border-current"}`}
											aria-hidden="true"
										/>
									)}
									<p className={`font-ui text-sm leading-snug ${isViewing ? "font-semibold" : ""}`}>
										{item.order}. {item.title}
									</p>
								</a>
							</li>
						);
					})}
				</ol>

				{nextLesson && (
					<a href={`/cursos/${courseSlug}/${nextLesson.order}`} className="btn btn-primary mt-5 w-full">
						Siguiente lección →
					</a>
				)}
			</aside>
		</div>
	);
}
