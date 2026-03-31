import SwiftUI
import HamAppServices
import HamCore

struct MenuBarHamsterGlyph: View {
    let state: MenuBarHamsterState
    let animationSpeed: Double
    let reduceMotion: Bool
    let hamsterSkin: String
    let hat: String

    var body: some View {
        let frame = PixelHamsterLibrary.frame(for: spriteState, variant: hamsterSkin, frameIndex: 0)
        let nsImage = PixelHamsterLibrary.renderToNSImage(frame: frame, hat: hat, variant: hamsterSkin, size: NSSize(width: 18, height: 18))
        Image(nsImage: nsImage)
    }

    private var spriteState: HamsterSpriteState {
        switch state {
        case .idle:
            return .idle
        case .running:
            return .run
        case .waiting:
            return .alert
        case .error:
            return .error
        case .done:
            return .idle
        }
    }
}

// MARK: - Grid Cell

private struct GridCell: Identifiable {
    let occupant: PixelOfficeOccupant
    let centerX: CGFloat
    let centerY: CGFloat
    let cellWidth: CGFloat

    var id: String { occupant.id }
}

// MARK: - Pixel Office View

struct PixelOfficeView: View {
    let occupants: [PixelOfficeOccupant]
    let animationSpeedMultiplier: Double
    let reduceMotion: Bool
    let hamsterSkin: String
    let hat: String
    let deskTheme: String
    var onSelectAgent: ((String) -> Void)? = nil

    @State private var hoveredAgentID: String?

    private let wallHeight: CGFloat = 50
    private let rowHeight: CGFloat = 110
    private let colCount = 3

    private var rowCount: Int {
        max(1, Int(ceil(Double(max(occupants.count, 1)) / Double(colCount))))
    }

    private var officeHeight: CGFloat {
        wallHeight + CGFloat(rowCount) * rowHeight
    }

    var body: some View {
        GeometryReader { geo in
            let officeWidth = geo.size.width
            ZStack(alignment: .topLeading) {
                // Background: wall + floor via Canvas
                Canvas { context, size in
                    drawBackground(context: context, size: size)
                }
                .frame(width: officeWidth, height: officeHeight)

                // Grid-placed hamsters
                let cells = gridCells(officeWidth: officeWidth)
                ForEach(cells) { cell in
                    hamsterCell(occupant: cell.occupant, cellWidth: cell.cellWidth)
                        .frame(width: cell.cellWidth, height: rowHeight)
                        .contentShape(Rectangle())
                        .onHover { isHovered in
                            hoveredAgentID = isHovered ? cell.occupant.agent.id : nil
                        }
                        .brightness(hoveredAgentID == cell.occupant.agent.id ? 0.08 : 0)
                        .animation(.easeInOut(duration: 0.15), value: hoveredAgentID)
                        .onTapGesture {
                            onSelectAgent?(cell.occupant.agent.id)
                        }
                        .position(x: cell.centerX, y: cell.centerY)
                }
            }
            .frame(width: officeWidth, height: officeHeight)
        }
        .frame(height: officeHeight)
        .clipShape(RoundedRectangle(cornerRadius: 8))
    }

    // MARK: - Grid layout

    private func gridCells(officeWidth: CGFloat) -> [GridCell] {
        guard !occupants.isEmpty else { return [] }

        let rows = Int(ceil(Double(occupants.count) / Double(colCount)))

        var cells: [GridCell] = []
        for (index, occupant) in occupants.enumerated() {
            let row = index / colCount
            let col = index % colCount

            // Last row: center remaining items
            let itemsInThisRow: Int
            if row == rows - 1 {
                itemsInThisRow = occupants.count - row * colCount
            } else {
                itemsInThisRow = colCount
            }

            let rowCellWidth = officeWidth / CGFloat(itemsInThisRow)
            let centerX = rowCellWidth * CGFloat(col) + rowCellWidth / 2
            let centerY = wallHeight + CGFloat(row) * rowHeight + rowHeight / 2

            cells.append(GridCell(
                occupant: occupant,
                centerX: centerX,
                centerY: centerY,
                cellWidth: rowCellWidth
            ))
        }
        return cells
    }

    // MARK: - Background Canvas

    private func drawBackground(context: GraphicsContext, size: CGSize) {
        // Wall area: soft blue-gray
        let wallRect = CGRect(x: 0, y: 0, width: size.width, height: wallHeight)
        context.fill(Path(wallRect), with: .color(Color(red: 0.85, green: 0.88, blue: 0.92)))

        // Baseboard (bottom of wall)
        let baseboardH: CGFloat = 6
        let baseboardRect = CGRect(x: 0, y: wallHeight - baseboardH, width: size.width, height: baseboardH)
        context.fill(Path(baseboardRect), with: .color(Color(red: 0.75, green: 0.78, blue: 0.82)))

        // Floor area: cool gray tile (contrasts with brown hamsters)
        let floorRect = CGRect(x: 0, y: wallHeight, width: size.width, height: size.height - wallHeight)
        context.fill(Path(floorRect), with: .color(Color(red: 0.28, green: 0.32, blue: 0.38)))

        // Floor tile grid
        let tileSize: CGFloat = 24
        let tileLineColor = Color(red: 0.24, green: 0.28, blue: 0.34)
        // Horizontal lines
        var py = wallHeight + tileSize
        while py < size.height {
            context.stroke(
                Path { p in p.move(to: CGPoint(x: 0, y: py)); p.addLine(to: CGPoint(x: size.width, y: py)) },
                with: .color(tileLineColor), lineWidth: 0.5
            )
            py += tileSize
        }
        // Vertical lines
        var px: CGFloat = tileSize
        while px < size.width {
            context.stroke(
                Path { p in p.move(to: CGPoint(x: px, y: wallHeight)); p.addLine(to: CGPoint(x: px, y: size.height)) },
                with: .color(tileLineColor), lineWidth: 0.5
            )
            px += tileSize
        }

        // Wall-floor border
        context.stroke(
            Path { p in p.move(to: CGPoint(x: 0, y: wallHeight)); p.addLine(to: CGPoint(x: size.width, y: wallHeight)) },
            with: .color(Color(red: 0.50, green: 0.52, blue: 0.56)), lineWidth: 2
        )

        // Wall decorations
        drawWallWindow(context: context, wallWidth: size.width)
        drawWallClock(context: context, wallWidth: size.width)
        drawWallWhiteboard(context: context, wallWidth: size.width)
        drawWallPoster(context: context, wallWidth: size.width)
    }

    private func drawWallWindow(context: GraphicsContext, wallWidth: CGFloat) {
        let p: CGFloat = 3
        let winW: CGFloat = p * 14
        let winH: CGFloat = p * 8
        let cx = wallWidth / 2 - winW / 2
        let cy: CGFloat = 4

        // Sky glass
        let glassRect = CGRect(x: cx, y: cy, width: winW, height: winH)
        context.fill(Path(glassRect), with: .color(Color(red: 0.60, green: 0.82, blue: 0.96).opacity(0.75)))

        // Wood frame (outer)
        let frameColor = Color(red: 0.55, green: 0.38, blue: 0.22)
        context.stroke(Path(glassRect), with: .color(frameColor), lineWidth: 2)

        // Cross divider: horizontal
        let hLine = Path { p2 in
            p2.move(to: CGPoint(x: cx, y: cy + winH / 2))
            p2.addLine(to: CGPoint(x: cx + winW, y: cy + winH / 2))
        }
        context.stroke(hLine, with: .color(frameColor), lineWidth: 1.5)

        // Cross divider: vertical
        let vLine = Path { p2 in
            p2.move(to: CGPoint(x: cx + winW / 2, y: cy))
            p2.addLine(to: CGPoint(x: cx + winW / 2, y: cy + winH))
        }
        context.stroke(vLine, with: .color(frameColor), lineWidth: 1.5)
    }

    private func drawWallClock(context: GraphicsContext, wallWidth: CGFloat) {
        let p: CGFloat = 3
        let winW: CGFloat = p * 14
        let cx = wallWidth / 2 + winW / 2 + p * 6
        let cy: CGFloat = wallHeight / 2
        let radius: CGFloat = p * 4

        // Clock face
        let clockRect = CGRect(x: cx - radius, y: cy - radius, width: radius * 2, height: radius * 2)
        context.fill(Path(ellipseIn: clockRect), with: .color(Color(red: 0.98, green: 0.96, blue: 0.90)))
        context.stroke(Path(ellipseIn: clockRect), with: .color(Color(red: 0.45, green: 0.30, blue: 0.18)), lineWidth: 1.5)

        // Hour hand (pointing to ~10)
        let hourAngle: CGFloat = -.pi / 2 + (.pi * 2 / 12) * 10
        let hourLen = radius * 0.55
        let hourEnd = CGPoint(x: cx + cos(hourAngle) * hourLen, y: cy + sin(hourAngle) * hourLen)
        let hourHand = Path { p2 in
            p2.move(to: CGPoint(x: cx, y: cy))
            p2.addLine(to: hourEnd)
        }
        context.stroke(hourHand, with: .color(Color(red: 0.20, green: 0.15, blue: 0.10)), lineWidth: 1.5)

        // Minute hand (pointing to ~2)
        let minAngle: CGFloat = -.pi / 2 + (.pi * 2 / 60) * 10
        let minLen = radius * 0.75
        let minEnd = CGPoint(x: cx + cos(minAngle) * minLen, y: cy + sin(minAngle) * minLen)
        let minHand = Path { p2 in
            p2.move(to: CGPoint(x: cx, y: cy))
            p2.addLine(to: minEnd)
        }
        context.stroke(minHand, with: .color(Color(red: 0.20, green: 0.15, blue: 0.10)), lineWidth: 1)
    }

    private func drawWallWhiteboard(context: GraphicsContext, wallWidth: CGFloat) {
        // Whiteboard on left side of wall
        let wbX: CGFloat = 16
        let wbY: CGFloat = 6
        let wbW: CGFloat = 50
        let wbH: CGFloat = 30
        // Board
        context.fill(Path(CGRect(x: wbX, y: wbY, width: wbW, height: wbH)), with: .color(.white.opacity(0.9)))
        context.stroke(Path(CGRect(x: wbX, y: wbY, width: wbW, height: wbH)), with: .color(Color(red: 0.6, green: 0.62, blue: 0.65)), lineWidth: 1.5)
        // Marker tray
        context.fill(Path(CGRect(x: wbX + 5, y: wbY + wbH, width: wbW - 10, height: 3)), with: .color(Color(red: 0.6, green: 0.62, blue: 0.65)))
        // Scribbles on whiteboard
        context.stroke(
            Path { p in p.move(to: CGPoint(x: wbX + 6, y: wbY + 8)); p.addLine(to: CGPoint(x: wbX + 30, y: wbY + 8)) },
            with: .color(Color.blue.opacity(0.4)), lineWidth: 1
        )
        context.stroke(
            Path { p in p.move(to: CGPoint(x: wbX + 6, y: wbY + 14)); p.addLine(to: CGPoint(x: wbX + 38, y: wbY + 14)) },
            with: .color(Color.red.opacity(0.35)), lineWidth: 1
        )
        context.stroke(
            Path { p in p.move(to: CGPoint(x: wbX + 6, y: wbY + 20)); p.addLine(to: CGPoint(x: wbX + 24, y: wbY + 20)) },
            with: .color(Color.green.opacity(0.4)), lineWidth: 1
        )
    }

    private func drawWallPoster(context: GraphicsContext, wallWidth: CGFloat) {
        // Small poster on right side of wall
        let pX = wallWidth - 70
        let pY: CGFloat = 8
        let pW: CGFloat = 36
        let pH: CGFloat = 28
        // Paper
        context.fill(Path(CGRect(x: pX, y: pY, width: pW, height: pH)), with: .color(Color(red: 0.98, green: 0.95, blue: 0.85)))
        context.stroke(Path(CGRect(x: pX, y: pY, width: pW, height: pH)), with: .color(Color(red: 0.7, green: 0.65, blue: 0.55)), lineWidth: 1)
        // Colored sticky notes
        let stickyColors: [Color] = [.yellow.opacity(0.6), .pink.opacity(0.5), .mint.opacity(0.5), .orange.opacity(0.5)]
        for i in 0..<4 {
            let sx = pX + 3 + CGFloat(i % 2) * 16
            let sy = pY + 3 + CGFloat(i / 2) * 12
            context.fill(Path(CGRect(x: sx, y: sy, width: 13, height: 10)), with: .color(stickyColors[i]))
        }
    }

    // MARK: - Hamster Cell View

    @ViewBuilder
    private func hamsterCell(occupant: PixelOfficeOccupant, cellWidth: CGFloat) -> some View {
        VStack(spacing: 0) {
            HStack(spacing: 4) {
                if occupant.agent.teamRole == "lead" {
                    Image(systemName: "crown.fill")
                        .font(.system(size: 9, weight: .bold))
                        .foregroundStyle(Color.yellow)
                } else if occupant.agent.teamRole == "teammate" {
                    Image(systemName: "person.2.fill")
                        .font(.system(size: 8, weight: .medium))
                        .foregroundStyle(.white.opacity(0.75))
                }

                if occupant.agent.teamTaskTotal > 0 {
                    Text("\(min(max(occupant.agent.teamTaskCompleted, 0), occupant.agent.teamTaskTotal))/\(occupant.agent.teamTaskTotal)")
                        .font(.system(size: 7, weight: .bold, design: .monospaced))
                        .foregroundStyle(.white.opacity(0.8))
                }
            }
            .frame(height: 10)

            // Status indicator (only waitingInput gets ❓, others use monitor glow)
            if occupant.agent.status == .waitingInput {
                Text("❓")
                    .font(.system(size: 10))
                    .padding(.bottom, 1)
            } else {
                Spacer().frame(height: 14)
            }

            // Main scene: sub-agents (back) + hamster (mid) + furniture (front)
            ZStack(alignment: .bottom) {
                // Layer 1 (back): Sub-agents in arc around main hamster
                if occupant.subAgentCount > 0 {
                    let count = min(occupant.subAgentCount, 6)
                    let radius: CGFloat = 38
                    // Arc from ~160° to ~20° (wider arc = more spacing)
                    let startAngle: CGFloat = .pi * 0.88
                    let endAngle: CGFloat = .pi * 0.12
                    ZStack {
                        // Render from center outward: center ones first (behind), edge ones last (front)
                        // Sort by distance from center: closest to center → lowest zIndex
                        ForEach(0..<count, id: \.self) { i in
                            let fraction = count == 1 ? 0.5 : CGFloat(i) / CGFloat(count - 1)
                            let angle = startAngle + (endAngle - startAngle) * fraction
                            let xOff = cos(angle) * radius
                            let yOff = -sin(angle) * radius * 0.55 - 30
                            // Distance from center (0.5) — farther from center = higher zIndex (in front)
                            let distFromCenter = abs(fraction - 0.5)
                            PixelHamsterSpriteView(
                                state: .run,
                                variant: hamsterSkin,
                                hat: "none",
                                animationSpeedMultiplier: animationSpeedMultiplier,
                                reduceMotion: reduceMotion
                            )
                            .frame(width: 16, height: 16)
                            .offset(x: xOff, y: yOff)
                            .zIndex(Double(distFromCenter * 10))
                        }
                        if occupant.subAgentCount > 6 {
                            Text("+\(occupant.subAgentCount - 6)")
                                .font(.system(size: 6, weight: .bold, design: .monospaced))
                                .foregroundStyle(.white.opacity(0.7))
                                .offset(x: radius + 6, y: -26)
                                .zIndex(10)
                        }
                    }
                }

                // Layer 2 (middle): Main hamster
                PixelHamsterSpriteView(
                    state: occupant.sprite,
                    variant: occupant.agent.avatarVariant == "default" ? hamsterSkin : occupant.agent.avatarVariant,
                    hat: hat,
                    animationSpeedMultiplier: animationSpeedMultiplier,
                    reduceMotion: reduceMotion
                )
                .frame(width: 32, height: 32)
                .offset(y: -14) // Hamster sits above the desk

                // Layer 3 (front): Furniture back-view (overlaps hamster feet)
                Canvas { context, size in
                    drawFrontProp(context: context, size: size, status: occupant.agent.status)
                }
                .frame(width: min(cellWidth - 8, 80), height: 30)
                .allowsHitTesting(false)
            }
            .frame(height: 62)

            // Name label (tight to desk)
            Text(occupant.agent.displayName)
                .font(.system(size: 8, weight: .medium, design: .monospaced))
                .foregroundStyle(.white.opacity(0.95))
                .lineLimit(1)
                .frame(maxWidth: cellWidth - 4)
                .offset(y: -10)
        }
    }

    // MARK: - Front Prop Canvas (furniture back-view, in front of hamster)

    private func drawFrontProp(context: GraphicsContext, size: CGSize, status: AgentStatus) {
        let cx = size.width / 2
        let deskColor = Color(red: 0.45, green: 0.35, blue: 0.25)
        let legColor = Color(red: 0.35, green: 0.28, blue: 0.20)
        let deskW: CGFloat = 56
        let dx = cx - deskW / 2

        // Common desk surface + legs
        context.fill(Path(CGRect(x: dx, y: 14, width: deskW, height: 3)), with: .color(deskColor))
        context.fill(Path(CGRect(x: dx, y: 17, width: deskW, height: 1)), with: .color(legColor))
        context.fill(Path(CGRect(x: dx + 3, y: 18, width: 3, height: 12)), with: .color(legColor))
        context.fill(Path(CGRect(x: dx + deskW - 6, y: 18, width: 3, height: 12)), with: .color(legColor))

        // Status-specific item ON the desk
        switch status {
        case .thinking, .runningTool, .booting:
            // iMac-style monitor back view + coffee mug
            let monW: CGFloat = 30
            let mx = cx - monW / 2
            let silver = Color(red: 0.78, green: 0.80, blue: 0.82)
            let silverDark = Color(red: 0.62, green: 0.64, blue: 0.66)
            // Panel back (taller, rounder)
            context.fill(Path(roundedRect: CGRect(x: mx, y: 0, width: monW, height: 14), cornerRadius: 2), with: .color(silver))
            // Bottom chin
            context.fill(Path(CGRect(x: mx, y: 12, width: monW, height: 2)), with: .color(silverDark))
            // Logo circle
            context.fill(Path(ellipseIn: CGRect(x: cx - 3, y: 3, width: 6, height: 6)), with: .color(silverDark.opacity(0.5)))
            // Short stand neck
            context.fill(Path(CGRect(x: cx - 2, y: 14, width: 4, height: 1)), with: .color(silverDark))
            // Stand base
            context.fill(Path(roundedRect: CGRect(x: cx - 8, y: 15, width: 16, height: 2), cornerRadius: 1), with: .color(silverDark))

            // Coffee mug (right side of desk)
            let mugX = cx + monW / 2 + 2
            let mugY: CGFloat = 8
            context.fill(Path(roundedRect: CGRect(x: mugX, y: mugY, width: 6, height: 7), cornerRadius: 1), with: .color(Color.white.opacity(0.85)))
            context.fill(Path(CGRect(x: mugX + 1, y: mugY + 1, width: 4, height: 3)), with: .color(Color.brown.opacity(0.6)))
            // Handle
            context.fill(Path(CGRect(x: mugX + 6, y: mugY + 2, width: 2, height: 4)), with: .color(Color.white.opacity(0.6)))

        case .reading:
            // Book stack (spines facing us)
            let bookColors: [Color] = [
                Color(red: 0.75, green: 0.20, blue: 0.20),
                Color(red: 0.20, green: 0.45, blue: 0.70),
                Color(red: 0.25, green: 0.60, blue: 0.35),
                Color(red: 0.80, green: 0.65, blue: 0.15)
            ]
            for i in 0..<4 {
                let bx = cx - 14 + CGFloat(i) * 7
                let bh: CGFloat = CGFloat(8 + (i % 2) * 3)
                context.fill(Path(CGRect(x: bx, y: 12 - bh, width: 6, height: bh)), with: .color(bookColors[i]))
            }

        case .error, .disconnected:
            // Monitor back with red glow spilling around edges
            let monW: CGFloat = 30
            let mx = cx - monW / 2
            let silver = Color(red: 0.78, green: 0.80, blue: 0.82)
            let silverDark = Color(red: 0.62, green: 0.64, blue: 0.66)
            // Red glow behind monitor
            context.fill(Path(roundedRect: CGRect(x: mx - 2, y: 0, width: monW + 4, height: 16), cornerRadius: 3), with: .color(Color.red.opacity(0.25)))
            // Panel
            context.fill(Path(roundedRect: CGRect(x: mx, y: 0, width: monW, height: 14), cornerRadius: 2), with: .color(silver))
            context.fill(Path(CGRect(x: mx, y: 12, width: monW, height: 2)), with: .color(silverDark))
            context.fill(Path(ellipseIn: CGRect(x: cx - 3, y: 3, width: 6, height: 6)), with: .color(Color.red.opacity(0.4)))
            // Stand
            context.fill(Path(CGRect(x: cx - 2, y: 14, width: 4, height: 1)), with: .color(silverDark))
            context.fill(Path(roundedRect: CGRect(x: cx - 8, y: 15, width: 16, height: 2), cornerRadius: 1), with: .color(silverDark))

        case .waitingInput:
            // Monitor back with orange glow
            let monW: CGFloat = 30
            let mx = cx - monW / 2
            let silver = Color(red: 0.78, green: 0.80, blue: 0.82)
            let silverDark = Color(red: 0.62, green: 0.64, blue: 0.66)
            // Orange glow behind monitor
            context.fill(Path(roundedRect: CGRect(x: mx - 2, y: 0, width: monW + 4, height: 16), cornerRadius: 3), with: .color(Color.orange.opacity(0.25)))
            // Panel
            context.fill(Path(roundedRect: CGRect(x: mx, y: 0, width: monW, height: 14), cornerRadius: 2), with: .color(silver))
            context.fill(Path(CGRect(x: mx, y: 12, width: monW, height: 2)), with: .color(silverDark))
            context.fill(Path(ellipseIn: CGRect(x: cx - 3, y: 3, width: 6, height: 6)), with: .color(Color.orange.opacity(0.4)))
            // Stand
            context.fill(Path(CGRect(x: cx - 2, y: 14, width: 4, height: 1)), with: .color(silverDark))
            context.fill(Path(roundedRect: CGRect(x: cx - 8, y: 15, width: 16, height: 2), cornerRadius: 1), with: .color(silverDark))

        case .idle, .sleeping:
            // Closed laptop on desk
            let lapW: CGFloat = 24
            let lx = cx - lapW / 2
            context.fill(Path(roundedRect: CGRect(x: lx, y: 8, width: lapW, height: 4), cornerRadius: 1), with: .color(Color(red: 0.30, green: 0.30, blue: 0.33)))
            // Small LED (sleeping)
            context.fill(Path(ellipseIn: CGRect(x: cx - 1, y: 9, width: 3, height: 2)), with: .color(Color.orange.opacity(0.4)))

        case .done:
            break
        }

        return
    }

}

// MARK: - Sprite View

private struct PixelHamsterSpriteView: View {
    let state: HamsterSpriteState
    let variant: String
    let hat: String
    let animationSpeedMultiplier: Double
    let reduceMotion: Bool

    var body: some View {
        TimelineView(.animation(minimumInterval: reduceMotion ? 1 : max(0.12, 0.45 / max(animationSpeedMultiplier, 0.25)))) { timeline in
            Canvas { context, size in
                let frame = PixelHamsterLibrary.frame(
                    for: state,
                    variant: variant,
                    frameIndex: reduceMotion ? 0 : PixelHamsterLibrary.frameIndex(for: timeline.date, state: state)
                )
                PixelHamsterLibrary.draw(frame: frame, hat: hat, variant: variant, in: context, size: size)
            }
        }
        .drawingGroup()
    }
}

// MARK: - Pixel Hamster Library

enum PixelHamsterLibrary {
    // Warm hamster palette
    private static let furDefault = Color(red: 0.76, green: 0.58, blue: 0.42)
    private static let furNight = Color(red: 0.55, green: 0.43, blue: 0.34)
    private static let furGolden = Color(red: 0.90, green: 0.75, blue: 0.35)
    private static let furMint = Color(red: 0.56, green: 0.78, blue: 0.68)
    private static let ear = Color(red: 0.95, green: 0.72, blue: 0.76)
    private static let cheek = Color(red: 0.96, green: 0.68, blue: 0.72)
    private static let belly = Color(red: 0.98, green: 0.95, blue: 0.88)
    private static let nose = Color(red: 0.92, green: 0.58, blue: 0.52)
    private static let eye = Color(red: 0.15, green: 0.12, blue: 0.12)
    private static let eyeShine = Color(red: 1.0, green: 1.0, blue: 1.0)
    private static let alert = Color(red: 1.0, green: 0.85, blue: 0.2)
    private static let error = Color(red: 0.88, green: 0.30, blue: 0.30)
    private static let success = Color(red: 0.35, green: 0.78, blue: 0.45)
    private static let paw = Color(red: 0.68, green: 0.50, blue: 0.38)
    private static let jacket = Color(red: 0.32, green: 0.36, blue: 0.48)
    private static let tie = Color(red: 0.85, green: 0.25, blue: 0.25)
    private static let shirt = Color(red: 0.95, green: 0.95, blue: 0.98)
    private static let line = Color(red: 0.82, green: 0.80, blue: 0.78)

    static func renderToNSImage(frame: [String], hat: String, variant: String, size: NSSize) -> NSImage {
        let image = NSImage(size: size, flipped: false) { rect in
            guard let cgContext = NSGraphicsContext.current?.cgContext else { return false }
            cgContext.setShouldAntialias(false)
            cgContext.setAllowsAntialiasing(false)
            cgContext.interpolationQuality = .none
            let rows = frame.count
            let columns = frame.isEmpty ? 0 : frame[0].count
            guard rows > 0, columns > 0 else { return true }
            let pixelSize = floor(min(rect.width / CGFloat(columns), rect.height / CGFloat(rows)))
            let xOffset = floor((rect.width - pixelSize * CGFloat(columns)) / 2)
            let yOffset = floor((rect.height - pixelSize * CGFloat(rows)) / 2)

            for (rowIndex, row) in frame.enumerated() {
                for (columnIndex, symbol) in row.enumerated() {
                    guard let nsColor = nsColor(for: symbol, variant: variant) else { continue }
                    cgContext.setFillColor(nsColor)
                    let y = floor(rect.height - yOffset - CGFloat(rowIndex + 1) * pixelSize)
                    let pixelRect = CGRect(
                        x: floor(xOffset + CGFloat(columnIndex) * pixelSize),
                        y: y,
                        width: pixelSize,
                        height: pixelSize
                    )
                    cgContext.fill(pixelRect)
                }
            }
            return true
        }
        image.isTemplate = false
        return image
    }

    private static func nsColor(for symbol: Character, variant: String) -> CGColor? {
        switch symbol {
        case "F":
            switch variant {
            case "golden": return CGColor(red: 0.90, green: 0.75, blue: 0.35, alpha: 1)
            case "mint":   return CGColor(red: 0.56, green: 0.78, blue: 0.68, alpha: 1)
            case "night":  return CGColor(red: 0.55, green: 0.43, blue: 0.34, alpha: 1)
            default:       return CGColor(red: 0.76, green: 0.58, blue: 0.42, alpha: 1)
            }
        case "E": return CGColor(red: 0.95, green: 0.72, blue: 0.76, alpha: 1)
        case "B": return CGColor(red: 0.98, green: 0.95, blue: 0.88, alpha: 1)
        case "K": return CGColor(red: 0.15, green: 0.12, blue: 0.12, alpha: 1)
        case "W": return CGColor(red: 1, green: 1, blue: 1, alpha: 1)
        case "C": return CGColor(red: 0.96, green: 0.68, blue: 0.72, alpha: 1)
        case "N": return CGColor(red: 0.92, green: 0.58, blue: 0.52, alpha: 1)
        case "P": return CGColor(red: 0.68, green: 0.50, blue: 0.38, alpha: 1)
        case "A": return CGColor(red: 1.0, green: 0.85, blue: 0.2, alpha: 1)
        case "R": return CGColor(red: 0.88, green: 0.30, blue: 0.30, alpha: 1)
        case "S": return CGColor(red: 0.35, green: 0.78, blue: 0.45, alpha: 1)
        case "J": return CGColor(red: 0.32, green: 0.36, blue: 0.48, alpha: 1)
        case "T": return CGColor(red: 0.85, green: 0.25, blue: 0.25, alpha: 1)
        case "H": return CGColor(red: 0.95, green: 0.95, blue: 0.98, alpha: 1)
        case "L": return CGColor(red: 0.82, green: 0.80, blue: 0.78, alpha: 1)
        default:  return nil
        }
    }

    static func frameIndex(for date: Date, state: HamsterSpriteState) -> Int {
        let frameCount = frames(for: state).count
        guard frameCount > 1 else { return 0 }
        return Int(date.timeIntervalSinceReferenceDate * 4).quotientAndRemainder(dividingBy: frameCount).remainder
    }

    static func frame(for state: HamsterSpriteState, variant: String, frameIndex: Int) -> [String] {
        let frames = frames(for: state)
        guard !frames.isEmpty else { return framesFor(.idle)[0] }
        _ = variant
        return frames[min(frameIndex, frames.count - 1)]
    }

    static func draw(frame: [String], hat: String, variant: String, in context: GraphicsContext, size: CGSize) {
        guard !frame.isEmpty else { return }
        let rows = frame.count
        let columns = frame[0].count
        let pixelSize = floor(min(size.width / CGFloat(columns), size.height / CGFloat(rows)))
        let xOffset = floor((size.width - pixelSize * CGFloat(columns)) / 2)
        let yOffset = floor((size.height - pixelSize * CGFloat(rows)) / 2)

        for (rowIndex, row) in frame.enumerated() {
            for (columnIndex, symbol) in row.enumerated() {
                guard let color = color(for: symbol, variant: variant) else { continue }
                let rect = CGRect(
                    x: floor(xOffset + CGFloat(columnIndex) * pixelSize),
                    y: floor(yOffset + CGFloat(rowIndex) * pixelSize),
                    width: pixelSize,
                    height: pixelSize
                )
                context.fill(Path(rect), with: .color(color))
            }
        }
        drawHat(hat, in: context, size: size, columns: columns, pixelSize: pixelSize, xOffset: xOffset, yOffset: yOffset)
    }

    private static func color(for symbol: Character, variant: String = "default") -> Color? {
        switch symbol {
        case "F":
            switch variant {
            case "golden": return furGolden
            case "mint":   return furMint
            case "night":  return furNight
            default:       return furDefault
            }
        case "E": return ear
        case "B": return belly
        case "K": return eye
        case "W": return eyeShine
        case "C": return cheek
        case "N": return nose
        case "P": return paw
        case "A": return alert
        case "R": return error
        case "S": return success
        case "J": return jacket
        case "T": return tie
        case "H": return shirt
        case "L": return line
        default:  return nil
        }
    }

    private static func drawHat(_ hat: String, in context: GraphicsContext, size: CGSize, columns: Int, pixelSize: CGFloat, xOffset: CGFloat, yOffset: CGFloat) {
        let color: Color
        switch hat {
        case "cap":
            color = .red
        case "beanie":
            color = .blue
        default:
            return
        }
        let width = pixelSize * 5
        let rect = CGRect(x: xOffset + pixelSize * 2, y: yOffset, width: width, height: pixelSize * 1.5)
        context.fill(Path(roundedRect: rect, cornerRadius: pixelSize * 0.5), with: .color(color))
    }

    private static func frames(for state: HamsterSpriteState) -> [[String]] {
        framesFor(state)
    }

    // 16x16 chonky hamster — face is 2x scaled original, suit has detail
    private static func framesFor(_ state: HamsterSpriteState) -> [[String]] {
        switch state {
        case .idle:
            return [[
                "....EE....EE....",
                "....EE....EE....",
                "..FFFFFFFFFFFF..",
                "..FFFFFFFFFFFF..",
                "FFBBKKBBBBKKBBFF",
                "FFBBKKBBBBKKBBFF",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "JJJLLLLTLLLLLJJJ",
                "JJJJHHHTTHHHJJJJ",
                "JJJJJHHTTHHJJJJJ",
                "JJJJJJJTTJJJJJJJ",
                "JJJJJJJJJJJJJJJJ",
                "JJJJJJJJJJJJJJJJ",
            ], [
                "....EE....EE....",
                "....EE....EE....",
                "..FFFFFFFFFFFF..",
                "..FFFFFFFFFFFF..",
                "FFBBKKBBBBKKBBFF",
                "FFBBKKBBBBKKBBFF",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "JJJLLLLTLLLLLJJJ",
                "JJJJHHHTTHHHJJJJ",
                "JJJJJHHTTHHJJJJJ",
                "JJJJJJJTTJJJJJJJ",
                "JJJJJJJJJJJJJJJJ",
                "JJJJJJJJJJJJJJ..",
            ]]
        case .walk:
            return [[
                "....EE....EE....",
                "....EE....EE....",
                "..FFFFFFFFFFFF..",
                "..FFFFFFFFFFFF..",
                "FFBBKKBBBBKKBBFF",
                "FFBBKKBBBBKKBBFF",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "JJJLLLLTLLLLLJJJ",
                "JJJJHHHTTHHHJJJJ",
                "JJJJJHHTTHHJJJJJ",
                "JJJJJJJTTJJJJJJJ",
                "JJJJJJJJJJJJJJJJ",
                "JJJJJJJJJJJJJJJJ",
            ], [
                "....EE....EE....",
                "....EE....EE....",
                "..FFFFFFFFFFFF..",
                "..FFFFFFFFFFFF..",
                "FFBBKKBBBBKKBBFF",
                "FFBBKKBBBBKKBBFF",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "JJJLLLLTLLLLLJJJ",
                "JJJJHHHTTHHHJJJJ",
                "JJJJJHHTTHHJJJJJ",
                "JJJJJJJTTJJJJJJJ",
                "JJJJJJJJJJJJJJJJ",
                "..JJJJJJJJJJJJJJ",
            ]]
        case .run:
            return [[
                "................",
                "................",
                "....EE....EE....",
                "....EE....EE....",
                "..FFFFFFFFFFFF..",
                "..FFFFFFFFFFFF..",
                "FFBBKKBBBBKKBBFF",
                "FFBBKKBBBBKKBBFF",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "JJJLLLLTLLLLLJJJ",
                "JJJJHHHTTHHHJJJJ",
                "JJJJJHHTTHHJJJJJ",
                "JJJJJJJJJJJJJJJJ",
            ], [
                "....EE....EE....",
                "....EE....EE....",
                "..FFFFFFFFFFFF..",
                "..FFFFFFFFFFFF..",
                "FFBBKKBBBBKKBBFF",
                "FFBBKKBBBBKKBBFF",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "JJJLLLLTLLLLLJJJ",
                "JJJJHHHTTHHHJJJJ",
                "JJJJJHHTTHHJJJJJ",
                "JJJJJJJTTJJJJJJJ",
                "JJJJJJJJJJJJJJJJ",
                "JJJJJJJJJJJJJJJJ",
            ]]
        case .type:
            return [[
                "....EE....EE....",
                "....EE....EE....",
                "..FFFFFFFFFFFF..",
                "..FFFFFFFFFFFF..",
                "FFBBKKBBBBKKBBFF",
                "FFBBKKBBBBKKBBFF",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "JJJLLLLTLLLLLJJJ",
                "JJJJHHHTTHHHJJJJ",
                "JJJJJHHTTHHJJJJJ",
                "JJJJJJJTTJJJJJJJ",
                "..AAAAAAAAAAAA..",
                "..AAAAAAAAAAAA..",
            ], [
                "....EE....EE....",
                "....EE....EE....",
                "..FFFFFFFFFFFF..",
                "..FFFFFFFFFFFF..",
                "FFBBKKBBBBKKBBFF",
                "FFBBKKBBBBKKBBFF",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "JJJLLLLTLLLLLJJJ",
                "JJJJHHHTTHHHJJJJ",
                "JJJJJHHTTHHJJJJJ",
                "JJJJJJJTTJJJJJJJ",
                "AA..AAAAAAAA..AA",
                "AA..AAAAAAAA..AA",
            ]]
        case .read:
            return [[
                "....EE....EE....",
                "....EE....EE....",
                "..FFFFFFFFFFFF..",
                "..FFFFFFFFFFFF..",
                "FFBBKKBBBBKKBBFF",
                "FFBBKKBBBBKKBBFF",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "JJJJHHHTTHHHJJJJ",
                "JJJJAAAAAAJJJJJ",
                "JJAAAAAAAAAAJJJ",
                "..AAAAAAAAAAAA..",
                "..AAAAAAAAAAAA..",
                "................",
            ]]
        case .think:
            return [[
                "....EE....EE...A",
                "....EE....EE....",
                "..FFFFFFFFFFFF..",
                "..FFFFFFFFFFFF..",
                "FFBBKKBBBBKKBBFF",
                "FFBBKKBBBBKKBBFF",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "JJJLLLLTLLLLLJJJ",
                "JJJJHHHTTHHHJJJJ",
                "JJJJJHHTTHHJJJJJ",
                "JJJJJJJTTJJJJJJJ",
                "JJJJJJJJJJJJJJJJ",
                "JJJJJJJJJJJJJJJJ",
            ], [
                "....EE....EE....",
                "....EE....EE...A",
                "..FFFFFFFFFFFF..",
                "..FFFFFFFFFFFF..",
                "FFBBKKBBBBKKBBFF",
                "FFBBKKBBBBKKBBFF",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "JJJLLLLTLLLLLJJJ",
                "JJJJHHHTTHHHJJJJ",
                "JJJJJHHTTHHJJJJJ",
                "JJJJJJJTTJJJJJJJ",
                "JJJJJJJJJJJJJJJJ",
                "JJJJJJJJJJJJJJJJ",
            ]]
        case .sleep:
            return [[
                "....EE....EE....",
                "....EE....EE....",
                "..FFFFFFFFFFFF..",
                "..FFFFFFFFFFFF..",
                "FFBBKKBBBBKKBBFF",
                "FFBBKKBBBBKKBBFF",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "JJJLLLLTLLLLLJJJ",
                "JJJJHHHTTHHHJJJJ",
                "JJJJJHHTTHHJJJJJ",
                "JJJJJJJTTJJJJJJJ",
                "JJJJJJJJJJJJJJJJ",
                "JJJJJJJJJJJJJJJJ",
            ], [
                "....EE....EE...A",
                "....EE....EE....",
                "..FFFFFFFFFFFF..",
                "..FFFFFFFFFFFF..",
                "FFBBKKBBBBKKBBFF",
                "FFBBKKBBBBKKBBFF",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "JJJLLLLTLLLLLJJJ",
                "JJJJHHHTTHHHJJJJ",
                "JJJJJHHTTHHJJJJJ",
                "JJJJJJJTTJJJJJJJ",
                "JJJJJJJJJJJJJJJJ",
                "JJJJJJJJJJJJJJJJ",
            ]]
        case .alert:
            return [[
                "....EE....EE...R",
                "....EE....EE....",
                "..FFFFFFFFFFFF..",
                "..FFFFFFFFFFFF..",
                "FFBBKKBBBBKKBBFF",
                "FFBBKKBBBBKKBBFF",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "JJJLLLLTLLLLLJJJ",
                "JJJJHHHTTHHHJJJJ",
                "JJJJJHHTTHHJJJJJ",
                "JJJJJJJTTJJJJJJJ",
                "JJJJJJJJJJJJJJJJ",
                "JJJJJJJJJJJJJJJJ",
            ], [
                "....EE....EE....",
                "....EE....EE...R",
                "..FFFFFFFFFFFF..",
                "..FFFFFFFFFFFF..",
                "FFBBKKBBBBKKBBFF",
                "FFBBKKBBBBKKBBFF",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "JJJLLLLTLLLLLJJJ",
                "JJJJHHHTTHHHJJJJ",
                "JJJJJHHTTHHJJJJJ",
                "JJJJJJJTTJJJJJJJ",
                "JJJJJJJJJJJJJJJJ",
                "JJJJJJJJJJJJJJJJ",
            ]]
        case .error:
            return [[
                "....EE....EE...R",
                "....EE....EE....",
                "..FFFFFFFFFFFF..",
                "..FFFFFFFFFFFF..",
                "FFBBKKBBBBKKBBFF",
                "FFBBKKBBBBKKBBFF",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "JJJLLLLTLLLLLJJJ",
                "JJJJHHHTTHHHJJJJ",
                "JJJJJHHTTHHJJJJJ",
                "JJJJJJJTTJJJJJJJ",
                "JJJJJJJJJJJJJJJJ",
                "JJJJJJJJJJJJJJJJ",
            ], [
                "....EE....EE....",
                "....EE....EE...R",
                "..FFFFFFFFFFFF..",
                "..FFFFFFFFFFFF..",
                "FFBBKKBBBBKKBBFF",
                "FFBBKKBBBBKKBBFF",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBNNNNBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "BBBBBBBBBBBBBBBB",
                "JJJLLLLTLLLLLJJJ",
                "JJJJHHHTTHHHJJJJ",
                "JJJJJHHTTHHJJJJJ",
                "JJJJJJJTTJJJJJJJ",
                "JJJJJJJJJJJJJJJJ",
                "JJJJJJJJJJJJJJJJ",
            ]]
        }
    }
}
