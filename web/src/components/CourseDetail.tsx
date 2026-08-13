import { useEffect, useState } from "react";
import { useStore } from "@nanostores/react";
import { $auth, initAuth } from "../stores/auth";
import { getCourse, type Course } from "../lib/api/courses";
import { listLessons, type LessonSummary } from "../lib/api/lessons";
import { ApiError } from "../lib/api/client";
import { CheckCircleIcon } from "./icons";

type Props = {
	slug: string;
};

/** Gates course content behind a session. The page is prerendered
 * statically (getStaticPaths in cursos/[slug].astro), so this is the only
 * place that knows whether the visitor is actually logged in — the HTML
 * itself never contains the course content for anonymous visitors. */
export default function CourseDetail({ slug }: Props) {
	const auth = useStore($auth);
	const [course, setCourse] = useState<Course | null>(null);
	const [lessons, setLessons] = useState<LessonSummary[] | null>(null);
	const [error, setError] = useState<string | null>(null);

	useEffect(() => {
		if (auth.status === "loading") void initAuth();
	}, [auth.status]);

	useEffect(() => {
		if (auth.status !== "authenticated") return;
		Promise.all([getCourse(slug), listLessons(slug)])
			.then(([courseData, lessonsData]) => {
				setCourse(courseData);
				setLessons(lessonsData);
			})
			.catch((err) => setError(err instanceof ApiError ? err.message : "No pudimos cargar el curso."));
	}, [auth.status, slug]);

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

	if (error) {
		return (
			<p role="alert" className="font-ui text-sm text-sacred-red">
				{error}
			</p>
		);
	}

	if (!course || !lessons) {
		return <span className="loading loading-spinner loading-md text-marian-blue" aria-label="Cargando curso" />;
	}

	return (
		<article>
			<h1 className="font-display text-3xl text-marian-blue-deep">{course.title}</h1>
			{course.description && <p className="mt-4 font-serif text-ink-80">{course.description}</p>}

			{lessons.length > 0 && (
				<div className="mt-10">
					<h2 className="font-display text-xl font-semibold text-marian-blue-deep">Lecciones</h2>
					<ol className="mt-4 flex flex-col gap-1">
						{lessons.map((lesson) => (
							<li key={lesson.order}>
								<a
									href={`/cursos/${course.slug}/${lesson.order}`}
									className="flex items-center gap-3 rounded-[4px] px-3 py-2.5 transition-colors hover:bg-paper-warm"
								>
									{lesson.completed ? (
										<CheckCircleIcon className="h-5 w-5 shrink-0 text-liturgical-gold" />
									) : (
										<span className="h-5 w-5 shrink-0 rounded-full border border-current" aria-hidden="true" />
									)}
									<span className="flex-1 font-ui text-sm text-ink-80">
										{lesson.order}. {lesson.title}
									</span>
								</a>
							</li>
						))}
					</ol>
				</div>
			)}
		</article>
	);
}
