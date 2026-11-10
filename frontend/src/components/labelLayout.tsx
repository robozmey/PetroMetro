export type LabelInput = {
    id: number
    stationX: number
    stationY: number
    width: number
    height: number
    originalX: number
    originalY: number
    originalRect: Rect
    baselineOffsetY: number
}

export type LabelPlacement = {
    id: number
    x: number
    y: number
    anchor: 'start' | 'end' | 'middle'
}

type Rect = {
    x: number
    y: number
    width: number
    height: number
}

type StationObstacle = {
    id: number
    x: number
    y: number
    radius: number
}

type LineSegment = {
    x1: number
    y1: number
    x2: number
    y2: number
}

type Candidate = {
    rect: Rect
    placement: LabelPlacement
}

const DOT_RADIUS = 2
const LABEL_GAP = 3
const PADDING = 1.2
const ITERATIONS = 10

// Площадь пересечения двух прямоугольников с небольшим отступом.
function overlapArea(a: Rect, b: Rect): number {
    const left = Math.max(
        a.x - PADDING,
        b.x - PADDING,
    )
    const right = Math.min(
        a.x + a.width + PADDING,
        b.x + b.width + PADDING,
    )
    const top = Math.max(
        a.y - PADDING,
        b.y - PADDING,
    )
    const bottom = Math.min(
        a.y + a.height + PADDING,
        b.y + b.height + PADDING,
    )

    return Math.max(0, right - left) * Math.max(0, bottom - top)
}

function intersectsCircle(rect: Rect, circle: StationObstacle): boolean {
    const nearestX = Math.max(
        rect.x,
        Math.min(circle.x, rect.x + rect.width),
    )
    const nearestY = Math.max(
        rect.y,
        Math.min(circle.y, rect.y + rect.height),
    )

    const dx = circle.x - nearestX
    const dy = circle.y - nearestY
    const radius = circle.radius + PADDING

    return dx * dx + dy * dy < radius * radius
}

// Проверка пересечения отрезка с прямоугольником.
function intersectsLine(rect: Rect, line: LineSegment): boolean {
    const left = rect.x - PADDING
    const right = rect.x + rect.width + PADDING
    const top = rect.y - PADDING
    const bottom = rect.y + rect.height + PADDING

    const dx = line.x2 - line.x1
    const dy = line.y2 - line.y1

    let tMin = 0
    let tMax = 1

    const p = [-dx, dx, -dy, dy]
    const q = [
        line.x1 - left,
        right - line.x1,
        line.y1 - top,
        bottom - line.y1,
    ]

    for (let i = 0; i < 4; i++) {
        if (p[i] === 0) {
            if (q[i] < 0) return false
            continue
        }

        const t = q[i] / p[i]

        if (p[i] < 0) {
            tMin = Math.max(tMin, t)
        } else {
            tMax = Math.min(tMax, t)
        }

        if (tMin > tMax) return false
    }

    return true
}

function makeCandidate(
    label: LabelInput,
    rect: Rect,
    anchor: 'start' | 'end' | 'middle',
): Candidate {
    let x = rect.x

    if (anchor === 'middle') {
        x += label.width / 2
    } else if (anchor === 'end') {
        x += label.width
    }

    return {
        rect,
        placement: {
            id: label.id,
            x,
            y: rect.y - label.baselineOffsetY,
            anchor,
        },
    }
}

// Исх положение + варианты вокруг станции.
function getCandidates(label: LabelInput): Candidate[] {
    const { stationX, stationY, width, height } = label
    const gap = DOT_RADIUS + LABEL_GAP

    const centeredX = stationX - width / 2
    const centeredY = stationY - height / 2

    const candidates = [
        makeCandidate(label, label.originalRect, 'middle'),

        // Сверху и снизу.
        makeCandidate(
            label,
            {
                x: centeredX,
                y: stationY - gap - height,
                width,
                height,
            },
            'middle',
        ),
        makeCandidate(
            label,
            {
                x: centeredX,
                y: stationY + gap,
                width,
                height,
            },
            'middle',
        ),

        // Слева и справа.
        makeCandidate(
            label,
            {
                x: stationX - gap - width,
                y: centeredY,
                width,
                height,
            },
            'end',
        ),
        makeCandidate(
            label,
            {
                x: stationX + gap,
                y: centeredY,
                width,
                height,
            },
            'start',
        ),

        // Смещение вдоль горизонтали.
        ...[-1, 1].flatMap((direction) => [
            makeCandidate(
                label,
                {
                    x: centeredX + direction * width * 0.6,
                    y: stationY - gap - height,
                    width,
                    height,
                },
                'middle',
            ),
            makeCandidate(
                label,
                {
                    x: centeredX + direction * width * 0.6,
                    y: stationY + gap,
                    width,
                    height,
                },
                'middle',
            ),
        ]),

        // Смещение вдоль вертикали.
        ...[-1, 1].flatMap((direction) => [
            makeCandidate(
                label,
                {
                    x: stationX - gap - width,
                    y: centeredY + direction * height * 1.5,
                    width,
                    height,
                },
                'end',
            ),
            makeCandidate(
                label,
                {
                    x: stationX + gap,
                    y: centeredY + direction * height * 1.5,
                    width,
                    height,
                },
                'start',
            ),
        ]),
    ]

    const directions = [
        [-1, -1], [0, -1], [1, -1],
        [-1, 0],              [1, 0],
        [-1, 1],  [0, 1],  [1, 1],
    ] as const

    for (const distance of [1, 2]) {
        for (const [directionX, directionY] of directions) {
            const extraGap = distance * (height + LABEL_GAP)
            const offsetX = directionX === 0
                ? 0
                : directionX * (gap + width / 2 + extraGap)
            const offsetY = directionY === 0
                ? 0
                : directionY * (gap + height / 2 + extraGap)

            candidates.push(
                makeCandidate(
                    label,
                    {
                        x: stationX + offsetX - width / 2,
                        y: stationY + offsetY - height / 2,
                        width,
                        height,
                    },
                    'middle',
                ),
            )
        }
    }

    // Убираем дублирующиеся варианты.
    const unique = new Map<string, Candidate>()

    for (const candidate of candidates) {
        const { x, y } = candidate.rect
        const key = `${x.toFixed(2)}:${y.toFixed(2)}:${candidate.placement.anchor}`
        unique.set(key, candidate)
    }

    return [...unique.values()]
}

function calculateCost(
    candidate: Candidate,
    label: LabelInput,
    otherRects: Rect[],
    stations: StationObstacle[],
    lines: LineSegment[],
): number {
    const rect = candidate.rect

    let cost = 0

    // Пересечение с др подписями
    for (const other of otherRects) {
        cost += overlapArea(rect, other) * 1000
    }

    // Пересечение с кружочками станций
    for (const station of stations) {
        if (station.id !== label.id && intersectsCircle(rect, station)) {
            cost += 1_000_000
        }
    }

    // Пересечение с линиями неоч
    for (const line of lines) {
        if (intersectsLine(rect, line)) {
            cost += 1
        }
    }

    // Штраф за удаление от исходного положения.
    const originalCenterX =
        label.originalRect.x + label.width / 2
    const originalCenterY =
        label.originalRect.y + label.height / 2

    const currentCenterX = rect.x + rect.width / 2
    const currentCenterY = rect.y + rect.height / 2

    const distance = Math.hypot(
        currentCenterX - originalCenterX,
        currentCenterY - originalCenterY,
    )

    cost += distance * 2

    return cost
}

export function arrangeLabels(
    labels: LabelInput[],
    stations: StationObstacle[],
    lines: LineSegment[],
) {
    const current = new Map<number, Candidate>()

    for (const label of labels) {
        current.set(
            label.id,
            makeCandidate(label, label.originalRect, 'middle'),
        )
    }

    const sortedLabels = [...labels].sort((a, b) => a.id - b.id)

    for (let iteration = 0; iteration < ITERATIONS; iteration++) {
        let changed = false

        for (const label of sortedLabels) {
            const otherRects = labels
                .filter((other) => other.id !== label.id)
                .map((other) => current.get(other.id)?.rect)
                .filter((rect): rect is Rect => rect !== undefined)

            const candidates = getCandidates(label)

            let best = current.get(label.id)!
            let bestCost = calculateCost(
                best,
                label,
                otherRects,
                stations,
                lines,
            )

            for (const candidate of candidates) {
                const cost = calculateCost(
                    candidate,
                    label,
                    otherRects,
                    stations,
                    lines,
                )

                if (cost < bestCost - 0.001) {
                    best = candidate
                    bestCost = cost
                }
            }

            if (best !== current.get(label.id)) {
                current.set(label.id, best)
                changed = true
            }
        }

        if (!changed) break
    }

    return {
        placements: labels.map(
            (label) => current.get(label.id)!.placement,
        ),
    }
}
