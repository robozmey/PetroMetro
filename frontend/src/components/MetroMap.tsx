import { useEffect, useLayoutEffect, useRef, useState } from 'react'

import YearTimeline, {
  CITY_YEAR_RANGES,
  type MetroCitySlug,
} from './YearTimeline'
import type { MetroSchema, Station } from '../types/metro'
import './MetroMap.css'
import { arrangeLabels, type LabelPlacement } from './labelLayout'

type MetroMapProps = {
  schema: MetroSchema
  city: MetroCitySlug
}

export function MetroMap({ schema, city }: MetroMapProps) {
  const [hoveredStation, setHoveredStation] = useState<Station | null>(null)
  const [selectedYear, setSelectedYear] = useState(
    CITY_YEAR_RANGES[city].end,
  )
  const svgRef = useRef<SVGSVGElement>(null)
  const [labelPlacements, setLabelPlacements] = useState<LabelPlacement[]>([])

  useEffect(() => {
    setSelectedYear(CITY_YEAR_RANGES[city].end)
    setHoveredStation(null)
  }, [city])

  const range = CITY_YEAR_RANGES[city]

  const year = Math.max(
    range.start,
    Math.min(range.end, selectedYear),
  )

  const visibleStations = schema.stations.filter(
    (station) => station.year == null || station.year <= year,
  )
  const stationById = new Map(visibleStations.map((station) => [station.id, station]),)
  useLayoutEffect(() => {
    const svg = svgRef.current
    if (!svg) return

    const textElements = svg.querySelectorAll<SVGTextElement>(
      '.station__label',
    )

    const textById = new Map(
      Array.from(textElements).map((element) => [
        Number(element.dataset.stationId),
        element,
      ]),
    )

    const labels = visibleStations.flatMap((station) => {
      const element = textById.get(station.id)
      if (!element) return []

      const bounds = element.getBBox()
      const originalX = station.x
      const originalY = station.y + 5.6
      const baselineOffsetY =
        bounds.y - Number(element.getAttribute('y'))

      return [{
        id: station.id,
        stationX: station.x,
        stationY: station.y,
        width: bounds.width,
        height: bounds.height,
        originalX,
        originalY,
        originalRect: {
          x: originalX - bounds.width / 2,
          y: originalY + baselineOffsetY,
          width: bounds.width,
          height: bounds.height,
        },
        baselineOffsetY,
      }]
    })

    const stationObstacles = visibleStations.map((station) => ({
      id: station.id,
      x: station.x,
      y: station.y,
      radius: 2,
    }))

    const lineSegments = schema.lines.flatMap((line) => {
      const points = line.stations
        .map((id) => stationById.get(id))
        .filter((station): station is Station => station !== undefined)

      return points.slice(1).map((point, index) => ({
        x1: points[index].x,
        y1: points[index].y,
        x2: point.x,
        y2: point.y,
      }))
    })

    const result = arrangeLabels(
      labels,
      stationObstacles,
      lineSegments,
    )

    setLabelPlacements(result.placements)
  }, [schema, year])

  const placementById = new Map(
    labelPlacements.map((placement) => [
      placement.id,
      placement,
    ]),
  )
  return (
    <div className="metro-map-layout">
      <YearTimeline
        city={city}
        year={year}
        onYearChange={(nextYear: number) => {
          setSelectedYear(nextYear)
          setHoveredStation(null)
        }}
      />

      <svg
        ref={svgRef}
        className="metro-map"
        viewBox="0 0 700 420"
        role="img"
        aria-label="Схема Санкт-Петербургского метро"
        onPointerOver={(event) => {
          if (!(event.target instanceof Element)) return

          const group = event.target.closest<SVGGElement>('[data-station-id]')
          if (!group) return

          const stationId = Number(group.dataset.stationId)
          const station = visibleStations.find((item) => item.id === stationId,)
          if (station) setHoveredStation(station)
        }}
        onPointerLeave={() => setHoveredStation(null)}
      >
        {city === 'saint-petersburg' && (
          <image
            className="metro-map__background"
            href="/maps/saint-petersburg-rivers.png"
            x="0"
            y="0"
            width="700"
            height="420"
            preserveAspectRatio="xMidYMid slice"
          />
        )}
        {schema.lines.map((line) => {
          const stations = line.stations
            .map((stationId) => stationById.get(stationId))
            .filter((station): station is Station => station !== undefined)

          return (
            <polyline
              key={line.id}
              points={stations.map(({ x, y }) => `${x},${y}`).join(' ')}
              fill="none"
              stroke={line.color}
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth="2.4"
            />
          )
        })}

        {visibleStations.map((station) => {
          const placement = placementById.get(station.id)

          return (
            <g
              key={station.id}
              className="station"
              data-station-id={station.id}
            >
              <circle cx={station.x} cy={station.y} r="2" />

              <text
                className="station__label"
                data-station-id={station.id}
                x={placement?.x ?? station.x}
                y={placement?.y ?? station.y + 5.6}
                textAnchor={placement?.anchor ?? 'middle'}
              >
                {station.name}
              </text>
            </g>
          )
        })}

        {hoveredStation && (
          <foreignObject
            className="station__tooltip"
            x={hoveredStation.x > 460 ? hoveredStation.x - 238 : hoveredStation.x + 18}
            y={Math.max(10, Math.min(hoveredStation.y - 45, 280))}
            width="220"
            height="130"
            onPointerLeave={() => setHoveredStation(null)}
          >
            <div className="station__tooltip-content">
              <strong>{hoveredStation.name}</strong>
              <p>
                Линии: {schema.lines
                  .filter((line) => line.stations.includes(hoveredStation.id))
                  .map((line) => line.name)
                  .join(', ') || 'не указаны'}
              </p>
              <p>Год открытия: {hoveredStation.year || "не указан"}.</p>
              <p>Здесь можно разместить описание станции.</p>
            </div>
          </foreignObject>
        )}
      </svg>
    </div>
  )
}
