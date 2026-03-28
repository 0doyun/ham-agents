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
            return .celebrate
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

    private let officeWidth: CGFloat = 420
    private let officeHeight: CGFloat = 220

    var body: some View {
        ZStack(alignment: .topLeading) {
            // Background: floor + walls + furniture
            Canvas { context, size in
                OfficeCanvasRenderer.draw(theme: deskTheme, in: context, size: size)
            }
            .frame(width: officeWidth, height: officeHeight)

            // Hamsters placed near their area's furniture
            ForEach(groupedOccupants, id: \.area) { group in
                ForEach(Array(group.occupants.enumerated()), id: \.element.id) { index, occupant in
                    let pos = hamsterPosition(area: group.area, index: index, total: group.occupants.count)
                    hamsterView(occupant: occupant)
                        .position(x: pos.x, y: pos.y)
                }
            }
        }
        .frame(width: officeWidth, height: officeHeight)
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
            case .check:    return ("✅", .green)
            }
        }()
        Text(symbol)
            .font(.system(size: 10))
            .foregroundStyle(color)
    }

    // MARK: - Layout

    /// Furniture anchor points for each area (center of the hamster cluster).
    private func furnitureAnchor(for area: OfficeArea) -> CGPoint {
        switch area {
        case .desk:       return CGPoint(x: 100, y: 155)  // left side, desk area
        case .bookshelf:  return CGPoint(x: 340, y: 100)  // right side, upper
        case .sofa:       return CGPoint(x: 330, y: 185)  // right side, lower
        case .alertLight: return CGPoint(x: 80, y: 80)    // left side, upper (near alert light)
        }
    }

    private func hamsterPosition(area: OfficeArea, index: Int, total: Int) -> CGPoint {
        let anchor = furnitureAnchor(for: area)
        let spacing: CGFloat = 50
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

        // Draw furniture
        drawDesk(in: context, size: size, p: p, palette: palette)
        drawBookshelf(in: context, size: size, p: p, palette: palette)
        drawSofa(in: context, size: size, p: p, palette: palette)
        drawAlertLight(in: context, size: size, p: p, palette: palette)
    }

    // Desk: left side, lower area
    private static func drawDesk(in context: GraphicsContext, size: CGSize, p: CGFloat, palette: ThemePalette) {
        let deskX = p * 6
        let deskY = size.height - p * 18
        let deskW = p * 22

        // Desk surface
        context.fill(Path(CGRect(x: deskX, y: deskY, width: deskW, height: p * 3)), with: .color(palette.furniture))
        // Desk legs
        context.fill(Path(CGRect(x: deskX + p * 1, y: deskY + p * 3, width: p * 2, height: p * 6)), with: .color(palette.furnitureDark))
        context.fill(Path(CGRect(x: deskX + deskW - p * 3, y: deskY + p * 3, width: p * 2, height: p * 6)), with: .color(palette.furnitureDark))

        // Monitor on desk
        let monX = deskX + p * 6
        let monY = deskY - p * 7
        context.fill(Path(CGRect(x: monX, y: monY, width: p * 10, height: p * 6)), with: .color(palette.screen))
        context.fill(Path(CGRect(x: monX + p, y: monY + p, width: p * 8, height: p * 4)), with: .color(palette.screenGlow))
        // Monitor stand
        context.fill(Path(CGRect(x: monX + p * 4, y: monY + p * 6, width: p * 2, height: p * 1)), with: .color(palette.furnitureDark))

        // Plant on desk
        let plantX = deskX + p * 18
        context.fill(Path(CGRect(x: plantX, y: deskY - p * 3, width: p * 2, height: p * 2)), with: .color(palette.plant))
        context.fill(Path(CGRect(x: plantX, y: deskY - p * 1, width: p * 2, height: p * 1)), with: .color(palette.pot))
    }

    // Bookshelf: right side, upper area (against back wall)
    private static func drawBookshelf(in context: GraphicsContext, size: CGSize, p: CGFloat, palette: ThemePalette) {
        let shelfX = size.width - p * 30
        let shelfY = p * 4
        let shelfW = p * 14
        let shelfH = p * 24

        // Frame
        context.fill(Path(CGRect(x: shelfX, y: shelfY, width: shelfW, height: shelfH)), with: .color(palette.furnitureDark))

        // 3 rows of books
        let bookColors: [Color] = [palette.bookRed, palette.bookBlue, palette.bookGreen, palette.bookYellow]
        for row in 0..<3 {
            let rowY = shelfY + p * 2 + CGFloat(row) * p * 7
            // Shelf plank
            context.fill(Path(CGRect(x: shelfX + p, y: rowY + p * 5, width: shelfW - p * 2, height: p)), with: .color(palette.furniture))
            // Books
            for book in 0..<5 {
                let bx = shelfX + p * 2 + CGFloat(book) * p * 2.2
                let bh = p * (3 + CGFloat(book % 2))
                let color = bookColors[(row * 5 + book) % bookColors.count]
                context.fill(Path(CGRect(x: bx, y: rowY + p * 5 - bh, width: p * 1.5, height: bh)), with: .color(color))
            }
        }
    }

    // Sofa: right side, lower area
    private static func drawSofa(in context: GraphicsContext, size: CGSize, p: CGFloat, palette: ThemePalette) {
        let sofaX = size.width - p * 32
        let sofaY = size.height - p * 12
        let sofaW = p * 24
        let sofaH = p * 8

        // Sofa back
        context.fill(Path(CGRect(x: sofaX, y: sofaY - p * 2, width: sofaW, height: p * 3)), with: .color(palette.furniture))
        // Sofa seat
        context.fill(Path(roundedRect: CGRect(x: sofaX, y: sofaY, width: sofaW, height: sofaH), cornerRadius: p), with: .color(palette.rug))
        // Armrests
        context.fill(Path(CGRect(x: sofaX - p * 1, y: sofaY, width: p * 2, height: sofaH)), with: .color(palette.furniture))
        context.fill(Path(CGRect(x: sofaX + sofaW - p, y: sofaY, width: p * 2, height: sofaH)), with: .color(palette.furniture))
        // Sofa legs
        context.fill(Path(CGRect(x: sofaX + p * 1, y: sofaY + sofaH, width: p * 2, height: p * 2)), with: .color(palette.furnitureDark))
        context.fill(Path(CGRect(x: sofaX + sofaW - p * 3, y: sofaY + sofaH, width: p * 2, height: p * 2)), with: .color(palette.furnitureDark))
    }

    // Alert light: left side, upper area
    private static func drawAlertLight(in context: GraphicsContext, size: CGSize, p: CGFloat, palette: ThemePalette) {
        // Warning stripes on floor near alert area
        for i in 0..<5 {
            let x = p * 4 + CGFloat(i) * p * 4
            if i % 2 == 0 {
                context.fill(Path(CGRect(x: x, y: p * 15, width: p * 3, height: p * 2)), with: .color(palette.alertStripe))
            }
        }

        // Alert board on wall
        let boardX = p * 6
        let boardY = p * 2
        context.fill(Path(CGRect(x: boardX, y: boardY, width: p * 10, height: p * 8)), with: .color(palette.alertBoard))
        context.fill(Path(CGRect(x: boardX + p, y: boardY + p, width: p * 8, height: p * 6)), with: .color(palette.alertBoardInner))

        // Warning triangle on board
        let triCX = boardX + p * 5
        let triY = boardY + p * 2
        var tri = Path()
        tri.move(to: CGPoint(x: triCX, y: triY))
        tri.addLine(to: CGPoint(x: triCX - p * 2, y: triY + p * 3))
        tri.addLine(to: CGPoint(x: triCX + p * 2, y: triY + p * 3))
        tri.closeSubpath()
        context.fill(tri, with: .color(palette.alertYellow))

        // Siren light (blinking beacon)
        context.fill(Path(CGRect(x: p * 18, y: p * 3, width: p * 3, height: p * 3)), with: .color(palette.alertRed.opacity(0.6)))
        context.fill(Path(CGRect(x: p * 18.5, y: p * 3.5, width: p * 2, height: p * 2)), with: .color(palette.alertRed))
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
        case .celebrate:
            return [[
                "SS..EE......EE.S",
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
                "S.JJJJJJJJJJJJ.S",
            ], [
                "..S.EE......EES.",
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
                ".SJJJJJJJJJJJJS.",
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
