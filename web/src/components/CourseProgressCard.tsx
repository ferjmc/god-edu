import type { CourseProgress } from "../lib/api/courses";
import { formatLongDate } from "../lib/format";

export default function CourseProgressCard({ course }: { course: CourseProgress }) {
	const ctaLabel = course.progressPercent === 0 ? "Empezar →" : course.progressPercent === 100 ? "Repasar →" : "Continuar →";
	// Si todavía no está inscripto (enrolledAt ausente), el CTA manda al
	// detalle del curso — ahí está el botón "Inscribirme" — nunca directo a
	// una lección. Recién con enrolledAt presente tiene sentido saltar
	// directo a la próxima lección pendiente, como antes.
	const ctaHref = !course.enrolledAt
		? `/cursos/${course.slug}`
		: course.nextLessonOrder
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
				{course.enrolledAt && (
					<p className="mt-1 font-ui text-xs text-ink-60">Inscripto desde {formatLongDate(course.enrolledAt)}</p>
				)}
			</div>
			<a href={ctaHref} className="mt-4 inline-block font-ui text-sm text-marian-blue hover:text-marian-blue-deep">
				{ctaLabel}
			</a>
			{course.certificateCode && (
				<a
					href={`/certificados?code=${course.certificateCode}`}
					className="mt-1 block font-ui text-sm text-marian-blue hover:text-marian-blue-deep"
				>
					Ver certificado →
				</a>
			)}
		</article>
	);
}
