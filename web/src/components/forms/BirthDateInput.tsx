import { useState } from "react";

const MONTHS = [
	"enero",
	"febrero",
	"marzo",
	"abril",
	"mayo",
	"junio",
	"julio",
	"agosto",
	"septiembre",
	"octubre",
	"noviembre",
	"diciembre",
];

const CURRENT_YEAR = new Date().getFullYear();
const YEARS = Array.from({ length: 111 }, (_, i) => CURRENT_YEAR - i);

function daysInMonth(year: number, month: number): number {
	// El día 0 del mes siguiente es el último día de este. Sin año elegido
	// todavía se asume uno bisiesto, así el 29 de febrero sigue disponible.
	return new Date(year || 2000, month, 0).getDate();
}

function pad(n: number): string {
	return String(n).padStart(2, "0");
}

type Props = {
	/** Recibe "YYYY-MM-DD" cuando día, mes y año están elegidos, o "" mientras
	 * la fecha esté incompleta. */
	onChange: (value: string) => void;
	required?: boolean;
};

/** Selects de día / mes / año en español (orden día-mes-año). Para una fecha
 * de nacimiento es más cómodo que un calendario —no hay que retroceder
 * décadas mes a mes— y no suma ninguna dependencia de date picker. Emite el
 * mismo formato ISO que espera la API (ver birthDateLayout en auth.go). */
export default function BirthDateInput({ onChange, required }: Props) {
	const [day, setDay] = useState(0);
	const [month, setMonth] = useState(0);
	const [year, setYear] = useState(0);

	const maxDay = month ? daysInMonth(year, month) : 31;

	function update(nextDay: number, nextMonth: number, nextYear: number) {
		const max = nextMonth ? daysInMonth(nextYear, nextMonth) : 31;
		const clampedDay = nextDay > max ? 0 : nextDay;
		setDay(clampedDay);
		setMonth(nextMonth);
		setYear(nextYear);
		onChange(
			clampedDay && nextMonth && nextYear ? `${nextYear}-${pad(nextMonth)}-${pad(clampedDay)}` : "",
		);
	}

	return (
		<fieldset className="flex flex-col gap-1">
			<legend className="mb-1 font-ui text-sm text-ink-80">Fecha de nacimiento</legend>
			<div className="join w-full">
				<select
					aria-label="Día"
					required={required}
					value={day || ""}
					onChange={(e) => update(Number(e.target.value), month, year)}
					className="select join-item w-1/4"
				>
					<option value="" disabled>
						Día
					</option>
					{Array.from({ length: maxDay }, (_, i) => i + 1).map((d) => (
						<option key={d} value={d}>
							{d}
						</option>
					))}
				</select>
				<select
					aria-label="Mes"
					required={required}
					value={month || ""}
					onChange={(e) => update(day, Number(e.target.value), year)}
					className="select join-item w-2/4"
				>
					<option value="" disabled>
						Mes
					</option>
					{MONTHS.map((name, i) => (
						<option key={name} value={i + 1}>
							{name.charAt(0).toUpperCase() + name.slice(1)}
						</option>
					))}
				</select>
				<select
					aria-label="Año"
					required={required}
					value={year || ""}
					onChange={(e) => update(day, month, Number(e.target.value))}
					className="select join-item w-1/4"
				>
					<option value="" disabled>
						Año
					</option>
					{YEARS.map((y) => (
						<option key={y} value={y}>
							{y}
						</option>
					))}
				</select>
			</div>
		</fieldset>
	);
}
