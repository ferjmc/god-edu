import type { CourseProgress } from "../lib/api/courses";

export default function CourseProgressCard({ course }: { course: CourseProgress }) {
	const ctaLabel = course.progressPercent === 0 ? "Empezar →" : course.progressPercent === 100 ? "Repasar →" : "Continuar →";
	// Si hay una lección pendiente, saltamos directo a ella en vez de mandar
	// siempre a la portada del curso — así "Empezar"/"Continuar" arrancan la
	// próxima lección con un solo clic, como en cualquier LMS.
	const ctaHref = course.nextLessonOrder
		? `/cursos/${course.slug}/${course.nextLessonOrder}`
		: `/cursos/${course.slug}`;
	return (
		<article className="rounded-[4px] border border-ink-15 bg-paper p-6 shadow-gold-edge">
			<span className="mb-3 block h-0.5 w-8 bg-liturgical-gold" aria-hidden="true" />
			<h3 className="font-display text-xl font-semibold text-ink">{course.title}</h3>
			{course.description && <p className="mt-2 font-serif text-sm leading-relaxed text-ink-80">{course.description}</p>}
			<div className="mt-4">
				<progress className="progress progress-primary w-full" value={course.completedLessons} max={Math.max(course.totalLessons, 1)} />
				<p className="mt-1 font-ui text-xs text-ink-60">
					{course.completedLessons}/{course.totalLessons} lecciones · {course.progressPercent}%
				</p>
			</div>
			<a href={ctaHref} className="mt-4 inline-block font-ui text-sm text-marian-blue hover:text-marian-blue-deep">
				{ctaLabel}
			</a>
		</article>
	);
}
