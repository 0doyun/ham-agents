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

// MARK: - Pixel Office View (Single Office Canvas)

struct PixelOfficeView: View {
    let occupants: [PixelOfficeOccupant]
    let animationSpeedMultiplier: Double
    let reduceMotion: Bool
    let hamsterSkin: String
    let hat: String
    let deskTheme: String
    var onSelectAgent: ((String) -> Void)? = nil

    private let officeHeight: CGFloat = 240

    var body: some View {
        GeometryReader { geo in
            let officeWidth = geo.size.width
            ZStack(alignment: .topLeading) {
                // Background: floor + walls + furniture
                Canvas { context, size in
                    OfficeCanvasRenderer.draw(theme: deskTheme, in: context, size: size)
                }

                // Hamsters placed near their area's furniture (middle layer)
                ForEach(groupedOccupants, id: \.area) { group in
                    ForEach(Array(group.occupants.enumerated()), id: \.element.id) { index, occupant in
                        let pos = hamsterPosition(area: group.area, index: index, total: group.occupants.count, officeWidth: officeWidth)
                        hamsterView(occupant: occupant)
                            .position(x: pos.x, y: pos.y)
                            .onTapGesture {
                                onSelectAgent?(occupant.agent.id)
                            }
                    }
                }

                // Foreground furniture layer (in front of hamsters for depth)
                Canvas { context, size in
                    OfficeCanvasRenderer.drawForeground(theme: deskTheme, in: context, size: size)
                }
                .allowsHitTesting(false)
            }
        }
        .frame(height: officeHeight)
        .clipShape(RoundedRectangle(cornerRadius: 8))
    }

    // MARK: - Hamster Rendering

    @ViewBuilder
    private func hamsterView(occupant: PixelOfficeOccupant) -> some View {
        VStack(spacing: 0) {
            // Status icon above the hamster
            if let icon = PixelOfficeMapper.statusIcon(for: occupant.agent.status) {
                statusIconView(icon)
            }

            // Agent name
            Text(occupant.agent.displayName)
                .font(.system(size: 7, weight: .medium, design: .monospaced))
                .foregroundStyle(.white.opacity(0.85))
                .lineLimit(1)

            // Hamster sprite + mini hamsters row
            HStack(alignment: .bottom, spacing: 2) {
                PixelHamsterSpriteView(
                    state: occupant.sprite,
                    variant: occupant.agent.avatarVariant == "default" ? hamsterSkin : occupant.agent.avatarVariant,
                    hat: hat,
                    animationSpeedMultiplier: animationSpeedMultiplier,
                    reduceMotion: reduceMotion
                )
                .frame(width: 32, height: 32)

                // Mini hamsters for sub-agents
                if occupant.subAgentCount > 0 {
                    ForEach(0..<min(occupant.subAgentCount, 4), id: \.self) { _ in
                        PixelHamsterSpriteView(
                            state: .run,
                            variant: hamsterSkin,
                            hat: "none",
                            animationSpeedMultiplier: animationSpeedMultiplier,
                            reduceMotion: reduceMotion
                        )
                        .frame(width: 16, height: 16)
                    }
                    if occupant.subAgentCount > 4 {
                        Text("+\(occupant.subAgentCount - 4)")
                            .font(.system(size: 7, weight: .bold, design: .monospaced))
                            .foregroundStyle(.white.opacity(0.6))
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func statusIconView(_ icon: StatusIcon) -> some View {
        let (symbol, color): (String, Color) = {
            switch icon {
            case .question: return ("❓", .yellow)
            case .warning:  return ("⚠️", .red)
            }
        }()
        Text(symbol)
            .font(.system(size: 10))
            .foregroundStyle(color)
    }

    // MARK: - Layout

    /// Hamster anchor points — positioned IN FRONT of each furniture piece.
    /// These must stay in sync with OfficeCanvasRenderer's furniture positions.
    private func furnitureAnchor(for area: OfficeArea, officeWidth: CGFloat) -> CGPoint {
        let p: CGFloat = 3 // must match renderer pixel unit
        switch area {
        // Desk: furniture at left edge (deskX = p*2). Hamster sits in chair in front of desk.
        case .desk:       return CGPoint(x: p * 16, y: officeHeight - p * 4)
        // Bookshelf: furniture at right edge. Hamster stands in front.
        case .bookshelf:  return CGPoint(x: officeWidth - p * 11, y: p * 28)
        // Alert: furniture at top-left corner. Hamster stands below the dashboard.
        case .alertLight: return CGPoint(x: p * 11, y: p * 24)
        }
    }

    private func hamsterPosition(area: OfficeArea, index: Int, total: Int, officeWidth: CGFloat) -> CGPoint {
        let anchor = furnitureAnchor(for: area, officeWidth: officeWidth)
        let spacing: CGFloat = 44
        let offset = CGFloat(index) * spacing - CGFloat(total - 1) * spacing / 2
        return CGPoint(x: anchor.x + offset, y: anchor.y)
    }

    private struct AreaGroup: Identifiable {
        let area: OfficeArea
        let occupants: [PixelOfficeOccupant]
        var id: String { area.rawValue }
    }

    private var groupedOccupants: [AreaGroup] {
        var dict: [OfficeArea: [PixelOfficeOccupant]] = [:]
        for occupant in occupants {
            dict[occupant.area, default: []].append(occupant)
        }
        return dict.map { AreaGroup(area: $0.key, occupants: $0.value) }
            .sorted { $0.area.rawValue < $1.area.rawValue }
    }
}

// MARK: - Office Canvas Renderer (Single Unified Space)

private enum OfficeCanvasRenderer {
    static func draw(theme: String, in context: GraphicsContext, size: CGSize) {
        let p = CGFloat(3) // pixel unit
        let palette = themePalette(theme)

        // Floor
        context.fill(Path(CGRect(origin: .zero, size: size)), with: .color(palette.floor))

        // Floor grid
        let gridColor = palette.floorGrid
        for x in stride(from: CGFloat(0), through: size.width, by: p * 4) {
            context.fill(Path(CGRect(x: x, y: 0, width: 1, height: size.height)), with: .color(gridColor))
        }
        for y in stride(from: CGFloat(0), through: size.height, by: p * 4) {
            context.fill(Path(CGRect(x: 0, y: y, width: size.width, height: 1)), with: .color(gridColor))
        }

        // Back wall (top strip)
        context.fill(Path(CGRect(x: 0, y: 0, width: size.width, height: p * 10)), with: .color(palette.wall))
        // Wall-floor border
        context.fill(Path(CGRect(x: 0, y: p * 10, width: size.width, height: p * 1)), with: .color(palette.furnitureDark))

        // Draw wall decorations first (behind furniture)
        drawWallDecorations(in: context, size: size, p: p, palette: palette)

        // Draw furniture
        drawDesk(in: context, size: size, p: p, palette: palette)
        drawBookshelf(in: context, size: size, p: p, palette: palette)
        drawAlertLight(in: context, size: size, p: p, palette: palette)
    }

    /// Foreground layer drawn ON TOP of hamsters for sideview depth.
    /// Renders the lower desk edge, chair, and coffee machine so hamsters appear behind furniture.
    static func drawForeground(theme: String, in context: GraphicsContext, size: CGSize) {
        let p = CGFloat(3)
        let palette = themePalette(theme)

        // Desk front edge (lower part of desk surface, partially overlaps hamster feet)
        let deskX = p * 2
        let deskY = size.height - p * 14
        let deskW = p * 28
        context.fill(Path(CGRect(x: deskX, y: deskY, width: deskW, height: p * 2)), with: .color(palette.furniture.opacity(0.85)))
        // Desk legs (foreground)
        context.fill(Path(CGRect(x: deskX + p * 1, y: deskY + p * 2, width: p * 2, height: p * 12)), with: .color(palette.furnitureDark.opacity(0.7)))
        context.fill(Path(CGRect(x: deskX + deskW - p * 3, y: deskY + p * 2, width: p * 2, height: p * 12)), with: .color(palette.furnitureDark.opacity(0.7)))

        // Office chair (in front of desk, partially overlapping hamster)
        let chairX = p * 14
        let chairY = size.height - p * 10
        // Chair seat
        context.fill(Path(roundedRect: CGRect(x: chairX, y: chairY, width: p * 8, height: p * 3), cornerRadius: p), with: .color(palette.furnitureDark.opacity(0.6)))
        // Chair base
        context.fill(Path(CGRect(x: chairX + p * 3, y: chairY + p * 3, width: p * 2, height: p * 4)), with: .color(Color.gray.opacity(0.4)))
        // Chair wheels
        context.fill(Path(CGRect(x: chairX + p * 1, y: chairY + p * 7, width: p * 1.5, height: p * 1)), with: .color(Color.gray.opacity(0.3)))
        context.fill(Path(CGRect(x: chairX + p * 5.5, y: chairY + p * 7, width: p * 1.5, height: p * 1)), with: .color(Color.gray.opacity(0.3)))

        // Coffee machine (small foreground prop in center)
        let cmX = size.width / 2 - p * 2
        let cmY = size.height - p * 8
        context.fill(Path(roundedRect: CGRect(x: cmX, y: cmY, width: p * 5, height: p * 6), cornerRadius: p * 0.5), with: .color(palette.appliance.opacity(0.7)))
        context.fill(Path(CGRect(x: cmX + p, y: cmY + p, width: p * 3, height: p * 2)), with: .color(palette.screen.opacity(0.5)))
        // Coffee cup
        context.fill(Path(roundedRect: CGRect(x: cmX + p * 5.5, y: cmY + p * 3, width: p * 2, height: p * 2.5), cornerRadius: p * 0.3), with: .color(Color.white.opacity(0.6)))
    }

    // Desk: left edge, lower area — L-shaped desk with dual monitor, keyboard, mug
    private static func drawDesk(in context: GraphicsContext, size: CGSize, p: CGFloat, palette: ThemePalette) {
        let deskX = p * 2
        let deskY = size.height - p * 22
        let deskW = p * 28

        // Desk surface (thick top)
        context.fill(Path(CGRect(x: deskX, y: deskY, width: deskW, height: p * 2)), with: .color(palette.furniture))
        context.fill(Path(CGRect(x: deskX, y: deskY + p * 2, width: deskW, height: p * 1)), with: .color(palette.furnitureDark))
        // Desk legs
        context.fill(Path(CGRect(x: deskX + p * 1, y: deskY + p * 3, width: p * 2, height: p * 8)), with: .color(palette.furnitureDark))
        context.fill(Path(CGRect(x: deskX + deskW - p * 3, y: deskY + p * 3, width: p * 2, height: p * 8)), with: .color(palette.furnitureDark))
        // Cable management bar
        context.fill(Path(CGRect(x: deskX + p * 4, y: deskY + p * 7, width: deskW - p * 8, height: p * 1)), with: .color(palette.furnitureDark.opacity(0.5)))

        // Monitor (larger, with bezel)
        let monX = deskX + p * 4
        let monY = deskY - p * 9
        // Bezel
        context.fill(Path(CGRect(x: monX, y: monY, width: p * 12, height: p * 8)), with: .color(palette.screen))
        // Screen
        context.fill(Path(CGRect(x: monX + p, y: monY + p, width: p * 10, height: p * 6)), with: .color(palette.screenGlow))
        // Code lines on screen
        context.fill(Path(CGRect(x: monX + p * 2, y: monY + p * 2, width: p * 5, height: p * 0.8)), with: .color(palette.plant.opacity(0.7)))
        context.fill(Path(CGRect(x: monX + p * 2, y: monY + p * 3.5, width: p * 7, height: p * 0.8)), with: .color(palette.bookBlue.opacity(0.5)))
        context.fill(Path(CGRect(x: monX + p * 3, y: monY + p * 5, width: p * 4, height: p * 0.8)), with: .color(palette.bookRed.opacity(0.4)))
        // Monitor stand
        context.fill(Path(CGRect(x: monX + p * 4, y: monY + p * 8, width: p * 4, height: p * 1)), with: .color(palette.furnitureDark))
        context.fill(Path(CGRect(x: monX + p * 3, y: monY + p * 9, width: p * 6, height: p * 0.5)), with: .color(palette.furnitureDark))

        // Keyboard
        let kbX = deskX + p * 5
        let kbY = deskY - p * 1.5
        context.fill(Path(roundedRect: CGRect(x: kbX, y: kbY, width: p * 8, height: p * 2), cornerRadius: p * 0.5), with: .color(Color.gray.opacity(0.4)))
        // Key dots
        for col in 0..<3 {
            for row in 0..<1 {
                context.fill(Path(CGRect(x: kbX + p * 1 + CGFloat(col) * p * 2.5, y: kbY + p * 0.5 + CGFloat(row) * p * 1.2, width: p * 1.5, height: p * 0.8)), with: .color(Color.gray.opacity(0.6)))
            }
        }

        // Coffee mug
        let mugX = deskX + p * 20
        let mugY = deskY - p * 3
        context.fill(Path(roundedRect: CGRect(x: mugX, y: mugY, width: p * 2.5, height: p * 3), cornerRadius: p * 0.5), with: .color(Color.white.opacity(0.85)))
        // Mug handle
        context.fill(Path(CGRect(x: mugX + p * 2.5, y: mugY + p * 0.5, width: p * 1, height: p * 2)), with: .color(Color.white.opacity(0.6)))
        // Coffee inside
        context.fill(Path(CGRect(x: mugX + p * 0.3, y: mugY + p * 0.3, width: p * 1.9, height: p * 1)), with: .color(Color.brown.opacity(0.7)))

        // Plant
        let plantX = deskX + p * 24
        // Pot
        context.fill(Path(CGRect(x: plantX, y: deskY - p * 1.5, width: p * 3, height: p * 2)), with: .color(palette.pot))
        // Leaves (layered)
        context.fill(Path(CGRect(x: plantX - p * 0.5, y: deskY - p * 4, width: p * 2, height: p * 2.5)), with: .color(palette.plant))
        context.fill(Path(CGRect(x: plantX + p * 1.5, y: deskY - p * 5, width: p * 2, height: p * 3.5)), with: .color(palette.plant.opacity(0.8)))
        context.fill(Path(CGRect(x: plantX + p * 0.5, y: deskY - p * 3.5, width: p * 2, height: p * 2)), with: .color(palette.plant.opacity(0.9)))

        // Office chair (in front of desk)
        let chairX = deskX + p * 8
        let chairY = deskY + p * 11
        // Seat
        context.fill(Path(roundedRect: CGRect(x: chairX, y: chairY, width: p * 8, height: p * 3), cornerRadius: p), with: .color(palette.furnitureDark))
        // Backrest
        context.fill(Path(roundedRect: CGRect(x: chairX + p, y: chairY - p * 4, width: p * 6, height: p * 4), cornerRadius: p), with: .color(palette.furnitureDark.opacity(0.8)))
        // Chair base/wheels
        context.fill(Path(CGRect(x: chairX + p * 3, y: chairY + p * 3, width: p * 2, height: p * 2)), with: .color(Color.gray.opacity(0.5)))
        context.fill(Path(CGRect(x: chairX + p * 1, y: chairY + p * 5, width: p * 6, height: p * 1)), with: .color(Color.gray.opacity(0.4)))
    }

    // Bookshelf: right edge, against back wall — tall shelf with varied books, globe, lamp
    private static func drawBookshelf(in context: GraphicsContext, size: CGSize, p: CGFloat, palette: ThemePalette) {
        let shelfX = size.width - p * 20
        let shelfY = p * 2
        let shelfW = p * 18
        let shelfH = p * 28

        // Outer frame
        context.fill(Path(CGRect(x: shelfX, y: shelfY, width: shelfW, height: shelfH)), with: .color(palette.furnitureDark))
        // Inner back
        context.fill(Path(CGRect(x: shelfX + p, y: shelfY + p, width: shelfW - p * 2, height: shelfH - p * 2)), with: .color(palette.furniture.opacity(0.3)))

        // 4 rows of books
        let bookColors: [Color] = [palette.bookRed, palette.bookBlue, palette.bookGreen, palette.bookYellow, palette.bookRed.opacity(0.7)]
        for row in 0..<4 {
            let rowY = shelfY + p * 1.5 + CGFloat(row) * p * 6.5
            // Shelf plank
            context.fill(Path(CGRect(x: shelfX + p * 0.5, y: rowY + p * 5.5, width: shelfW - p, height: p * 0.8)), with: .color(palette.furniture))

            // Books — varied widths and heights
            if row < 3 {
                var bx = shelfX + p * 1.5
                for book in 0..<6 {
                    let bw = p * (1.2 + CGFloat(book % 3) * 0.3)
                    let bh = p * (3.5 + CGFloat((book + row) % 3) * 0.8)
                    let color = bookColors[(row * 6 + book) % bookColors.count]
                    context.fill(Path(CGRect(x: bx, y: rowY + p * 5.5 - bh, width: bw, height: bh)), with: .color(color))
                    bx += bw + p * 0.3
                }
            }
        }

        // Globe on top shelf (row 3)
        let globeX = shelfX + p * 4
        let globeY = shelfY + p * 1.5 + p * 6.5 * 3 + p * 2
        // Globe stand
        context.fill(Path(CGRect(x: globeX + p * 1.5, y: globeY + p * 2.5, width: p * 1, height: p * 1.5)), with: .color(palette.furnitureDark))
        // Globe sphere
        let globeR = p * 2
        context.fill(Path(ellipseIn: CGRect(x: globeX, y: globeY - p * 0.5, width: globeR * 2, height: globeR * 2)), with: .color(palette.bookBlue.opacity(0.6)))
        context.fill(Path(ellipseIn: CGRect(x: globeX + p * 0.5, y: globeY, width: p * 1.5, height: p * 1)), with: .color(palette.plant.opacity(0.5)))

        // Framed photo on top shelf
        let photoX = shelfX + p * 10
        let photoY = shelfY + p * 1.5 + p * 6.5 * 3 + p * 1
        context.fill(Path(CGRect(x: photoX, y: photoY, width: p * 4, height: p * 3.5)), with: .color(palette.furnitureDark))
        context.fill(Path(CGRect(x: photoX + p * 0.5, y: photoY + p * 0.5, width: p * 3, height: p * 2.5)), with: .color(palette.screenGlow.opacity(0.3)))
    }

    // Alert light: left side, upper area — dashboard board, siren, warning stripes
    private static func drawAlertLight(in context: GraphicsContext, size: CGSize, p: CGFloat, palette: ThemePalette) {
        // Warning stripes on floor (chevron pattern)
        for i in 0..<6 {
            let x = p * 2 + CGFloat(i) * p * 5
            let y = p * 14
            if i % 2 == 0 {
                context.fill(Path(CGRect(x: x, y: y, width: p * 3.5, height: p * 1.5)), with: .color(palette.alertYellow.opacity(0.5)))
            } else {
                context.fill(Path(CGRect(x: x, y: y, width: p * 3.5, height: p * 1.5)), with: .color(palette.alertStripe))
            }
        }

        // Alert dashboard on wall (larger, with frame)
        let boardX = p * 4
        let boardY = p * 1.5
        let boardW = p * 14
        let boardH = p * 8
        // Frame
        context.fill(Path(roundedRect: CGRect(x: boardX - p * 0.5, y: boardY - p * 0.5, width: boardW + p, height: boardH + p), cornerRadius: p * 0.5), with: .color(palette.furnitureDark))
        // Board background
        context.fill(Path(CGRect(x: boardX, y: boardY, width: boardW, height: boardH)), with: .color(palette.alertBoard))
        // Inner area
        context.fill(Path(CGRect(x: boardX + p, y: boardY + p, width: boardW - p * 2, height: boardH - p * 2)), with: .color(palette.alertBoardInner))

        // Warning triangle (centered on board)
        let triCX = boardX + boardW / 2
        let triY = boardY + p * 1.5
        var tri = Path()
        tri.move(to: CGPoint(x: triCX, y: triY))
        tri.addLine(to: CGPoint(x: triCX - p * 2.5, y: triY + p * 3.5))
        tri.addLine(to: CGPoint(x: triCX + p * 2.5, y: triY + p * 3.5))
        tri.closeSubpath()
        context.fill(tri, with: .color(palette.alertYellow))
        // Exclamation mark
        context.fill(Path(CGRect(x: triCX - p * 0.3, y: triY + p * 1, width: p * 0.6, height: p * 1.2)), with: .color(palette.alertBoard))
        context.fill(Path(ellipseIn: CGRect(x: triCX - p * 0.3, y: triY + p * 2.5, width: p * 0.6, height: p * 0.6)), with: .color(palette.alertBoard))

        // Siren beacon (mounted on wall, with glow)
        let sirenX = boardX + boardW + p * 3
        let sirenY = p * 2
        // Glow
        context.fill(Path(ellipseIn: CGRect(x: sirenX - p * 1.5, y: sirenY - p * 1, width: p * 6, height: p * 5)), with: .color(palette.alertRed.opacity(0.15)))
        // Beacon base
        context.fill(Path(CGRect(x: sirenX, y: sirenY + p * 2, width: p * 3, height: p * 1)), with: .color(palette.furnitureDark))
        // Beacon dome
        context.fill(Path(roundedRect: CGRect(x: sirenX + p * 0.3, y: sirenY, width: p * 2.4, height: p * 2.5), cornerRadius: p), with: .color(palette.alertRed))
        context.fill(Path(roundedRect: CGRect(x: sirenX + p * 0.8, y: sirenY + p * 0.3, width: p * 1.2, height: p * 1.5), cornerRadius: p * 0.5), with: .color(palette.alertRed.opacity(0.5)))
    }

    // Wall decorations: window, clock, poster
    private static func drawWallDecorations(in context: GraphicsContext, size: CGSize, p: CGFloat, palette: ThemePalette) {
        // Window (center of back wall)
        let winX = size.width * 0.38
        let winY = p * 1
        let winW = p * 16
        let winH = p * 8
        // Window frame
        context.fill(Path(CGRect(x: winX - p * 0.5, y: winY - p * 0.5, width: winW + p, height: winH + p)), with: .color(palette.furnitureDark))
        // Window panes (sky blue)
        context.fill(Path(CGRect(x: winX, y: winY, width: winW / 2 - p * 0.3, height: winH)), with: .color(Color(red: 0.6, green: 0.8, blue: 1.0).opacity(0.5)))
        context.fill(Path(CGRect(x: winX + winW / 2 + p * 0.3, y: winY, width: winW / 2 - p * 0.3, height: winH)), with: .color(Color(red: 0.6, green: 0.8, blue: 1.0).opacity(0.5)))
        // Cross bar
        context.fill(Path(CGRect(x: winX + winW / 2 - p * 0.3, y: winY, width: p * 0.6, height: winH)), with: .color(palette.furnitureDark))
        context.fill(Path(CGRect(x: winX, y: winY + winH / 2 - p * 0.3, width: winW, height: p * 0.6)), with: .color(palette.furnitureDark))
        // Sunlight reflection
        context.fill(Path(CGRect(x: winX + p * 1, y: winY + p * 1, width: p * 2, height: p * 1)), with: .color(Color.white.opacity(0.3)))

        // Clock (on wall, right of window)
        let clockX = size.width * 0.56
        let clockY = p * 2
        let clockR = p * 3
        // Clock face
        context.fill(Path(ellipseIn: CGRect(x: clockX, y: clockY, width: clockR * 2, height: clockR * 2)), with: .color(Color.white.opacity(0.85)))
        // Clock border
        context.stroke(Path(ellipseIn: CGRect(x: clockX, y: clockY, width: clockR * 2, height: clockR * 2)), with: .color(palette.furnitureDark), lineWidth: p * 0.5)
        // Clock hands
        let cx = clockX + clockR
        let cy = clockY + clockR
        // Hour hand
        var hour = Path()
        hour.move(to: CGPoint(x: cx, y: cy))
        hour.addLine(to: CGPoint(x: cx - p * 1, y: cy - p * 1.5))
        context.stroke(hour, with: .color(palette.furnitureDark), lineWidth: p * 0.4)
        // Minute hand
        var minute = Path()
        minute.move(to: CGPoint(x: cx, y: cy))
        minute.addLine(to: CGPoint(x: cx + p * 0.5, y: cy - p * 2))
        context.stroke(minute, with: .color(palette.furnitureDark), lineWidth: p * 0.3)
        // Center dot
        context.fill(Path(ellipseIn: CGRect(x: cx - p * 0.3, y: cy - p * 0.3, width: p * 0.6, height: p * 0.6)), with: .color(palette.furnitureDark))
    }

    // MARK: - Theme Palette

    struct ThemePalette {
        let floor, floorGrid, wall: Color
        let furniture, furnitureDark: Color
        let screen, screenGlow: Color
        let plant, pot: Color
        let bookRed, bookBlue, bookGreen, bookYellow: Color
        let rug: Color
        let appliance: Color
        let alertStripe, alertBoard, alertBoardInner, alertYellow, alertRed: Color
    }

    private static func themePalette(_ theme: String) -> ThemePalette {
        switch theme {
        case "night-shift":
            return ThemePalette(
                floor: Color(red: 0.12, green: 0.11, blue: 0.18),
                floorGrid: Color.white.opacity(0.04),
                wall: Color(red: 0.15, green: 0.14, blue: 0.22),
                furniture: Color(red: 0.28, green: 0.22, blue: 0.35),
                furnitureDark: Color(red: 0.18, green: 0.14, blue: 0.24),
                screen: Color(red: 0.15, green: 0.15, blue: 0.25),
                screenGlow: Color(red: 0.25, green: 0.35, blue: 0.55),
                plant: Color(red: 0.2, green: 0.45, blue: 0.3),
                pot: Color(red: 0.35, green: 0.25, blue: 0.2),
                bookRed: Color(red: 0.55, green: 0.2, blue: 0.25),
                bookBlue: Color(red: 0.2, green: 0.25, blue: 0.5),
                bookGreen: Color(red: 0.2, green: 0.4, blue: 0.25),
                bookYellow: Color(red: 0.55, green: 0.45, blue: 0.2),
                rug: Color(red: 0.3, green: 0.2, blue: 0.35),
                appliance: Color(red: 0.22, green: 0.2, blue: 0.28),
                alertStripe: Color(red: 0.5, green: 0.35, blue: 0.1),
                alertBoard: Color(red: 0.25, green: 0.18, blue: 0.15),
                alertBoardInner: Color(red: 0.15, green: 0.12, blue: 0.1),
                alertYellow: Color(red: 0.85, green: 0.65, blue: 0.15),
                alertRed: Color(red: 0.75, green: 0.2, blue: 0.2)
            )
        case "sunny":
            return ThemePalette(
                floor: Color(red: 0.95, green: 0.90, blue: 0.82),
                floorGrid: Color.black.opacity(0.04),
                wall: Color(red: 0.92, green: 0.88, blue: 0.80),
                furniture: Color(red: 0.72, green: 0.58, blue: 0.42),
                furnitureDark: Color(red: 0.55, green: 0.42, blue: 0.30),
                screen: Color(red: 0.3, green: 0.3, blue: 0.35),
                screenGlow: Color(red: 0.6, green: 0.8, blue: 0.95),
                plant: Color(red: 0.35, green: 0.65, blue: 0.3),
                pot: Color(red: 0.7, green: 0.45, blue: 0.3),
                bookRed: Color(red: 0.8, green: 0.3, blue: 0.3),
                bookBlue: Color(red: 0.3, green: 0.45, blue: 0.75),
                bookGreen: Color(red: 0.3, green: 0.6, blue: 0.35),
                bookYellow: Color(red: 0.85, green: 0.7, blue: 0.25),
                rug: Color(red: 0.85, green: 0.75, blue: 0.55),
                appliance: Color(red: 0.88, green: 0.86, blue: 0.82),
                alertStripe: Color(red: 0.9, green: 0.7, blue: 0.2),
                alertBoard: Color(red: 0.75, green: 0.6, blue: 0.45),
                alertBoardInner: Color(red: 0.95, green: 0.92, blue: 0.85),
                alertYellow: Color(red: 0.95, green: 0.75, blue: 0.15),
                alertRed: Color(red: 0.85, green: 0.25, blue: 0.2)
            )
        default: // classic
            return ThemePalette(
                floor: Color(red: 0.16, green: 0.18, blue: 0.22),
                floorGrid: Color.white.opacity(0.04),
                wall: Color(red: 0.20, green: 0.22, blue: 0.28),
                furniture: Color(red: 0.35, green: 0.28, blue: 0.22),
                furnitureDark: Color(red: 0.22, green: 0.18, blue: 0.14),
                screen: Color(red: 0.15, green: 0.18, blue: 0.22),
                screenGlow: Color(red: 0.3, green: 0.5, blue: 0.65),
                plant: Color(red: 0.25, green: 0.55, blue: 0.3),
                pot: Color(red: 0.5, green: 0.35, blue: 0.25),
                bookRed: Color(red: 0.7, green: 0.25, blue: 0.25),
                bookBlue: Color(red: 0.25, green: 0.35, blue: 0.65),
                bookGreen: Color(red: 0.25, green: 0.5, blue: 0.3),
                bookYellow: Color(red: 0.7, green: 0.6, blue: 0.2),
                rug: Color(red: 0.4, green: 0.3, blue: 0.25),
                appliance: Color(red: 0.3, green: 0.32, blue: 0.35),
                alertStripe: Color(red: 0.7, green: 0.5, blue: 0.1),
                alertBoard: Color(red: 0.3, green: 0.22, blue: 0.18),
                alertBoardInner: Color(red: 0.2, green: 0.16, blue: 0.14),
                alertYellow: Color(red: 0.9, green: 0.7, blue: 0.15),
                alertRed: Color(red: 0.8, green: 0.22, blue: 0.22)
            )
        }
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
                "FFBBFFBBBBFFBBFF",
                "FFBBFFBBBBFFBBFF",
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
                "FFBBFFBBBBFFBBFF",
                "FFBBFFBBBBFFBBFF",
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
                "......RRRR......",
                "......RRRR......",
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
            ], [
                ".....RRRRRR.....",
                ".....RRRRRR.....",
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
            ]]
        }
    }
}
