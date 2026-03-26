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

// MARK: - Pixel Office View

struct PixelOfficeView: View {
    let occupants: [PixelOfficeOccupant]
    let animationSpeedMultiplier: Double
    let reduceMotion: Bool
    let hamsterSkin: String
    let hat: String
    let deskTheme: String

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack(alignment: .top, spacing: 4) {
                zoneCard(.desk, icon: "keyboard", title: "Desk")
                zoneCard(.library, icon: "books.vertical", title: "Library")
            }
            HStack(alignment: .top, spacing: 4) {
                zoneCard(.kitchen, icon: "cup.and.saucer", title: "Kitchen")
                zoneCard(.alertCorner, icon: "exclamationmark.triangle", title: "Alert")
            }
        }
    }

    @ViewBuilder
    private func zoneCard(_ zone: PixelOfficeZone, icon: String, title: String) -> some View {
        let zoneOccupants = occupants.filter { $0.zone == zone }

        ZStack {
            // Pixel furniture background
            Canvas { context, size in
                PixelFurnitureRenderer.draw(zone: zone, theme: deskTheme, in: context, size: size)
            }

            // Room label top-left
            VStack {
                HStack {
                    HStack(spacing: 3) {
                        Image(systemName: icon)
                            .font(.system(size: 8))
                        Text(title)
                            .font(.system(size: 9, weight: .bold, design: .monospaced))
                    }
                    .foregroundStyle(.white.opacity(0.7))
                    .padding(.horizontal, 6)
                    .padding(.top, 5)
                    Spacer()
                }
                Spacer()
            }

            // Hamsters anchored to furniture position, left-aligned
            if !zoneOccupants.isEmpty {
                VStack(spacing: 0) {
                    Spacer()
                    HStack(alignment: .bottom, spacing: 6) {
                        ForEach(zoneOccupants) { occupant in
                            VStack(spacing: 1) {
                                Text(occupant.agent.displayName)
                                    .font(.system(size: 7, weight: .medium, design: .monospaced))
                                    .foregroundStyle(.white.opacity(0.85))
                                    .lineLimit(1)

                                PixelHamsterSpriteView(
                                    state: occupant.sprite,
                                    variant: occupant.agent.avatarVariant == "default" ? hamsterSkin : occupant.agent.avatarVariant,
                                    hat: hat,
                                    animationSpeedMultiplier: animationSpeedMultiplier,
                                    reduceMotion: reduceMotion
                                )
                                .frame(width: 32, height: 32)
                            }
                        }
                        Spacer()
                    }
                    .padding(.horizontal, 6)
                    .padding(.bottom, furnitureBottomPadding(for: zone))
                }
            }
        }
        .frame(maxWidth: .infinity, minHeight: 100)
        .clipShape(RoundedRectangle(cornerRadius: 6))
    }

    private func furnitureBottomPadding(for zone: PixelOfficeZone) -> CGFloat {
        switch zone {
        case .desk:
            return 26  // sit at desk height
        case .library:
            return 6   // stand on floor by bookshelf
        case .kitchen:
            return 30  // stand in front of counter (counter is tall)
        case .alertCorner:
            return 8   // stand on warning stripes
        }
    }
}

// MARK: - Pixel Furniture Renderer

private enum PixelFurnitureRenderer {
    static func draw(zone: PixelOfficeZone, theme: String, in context: GraphicsContext, size: CGSize) {
        let p = CGFloat(3) // pixel unit size
        let palette = themePalette(theme)

        // Floor
        context.fill(Path(CGRect(origin: .zero, size: size)), with: .color(palette.floor))

        // Floor grid pattern
        let gridColor = palette.floorGrid
        for x in stride(from: CGFloat(0), through: size.width, by: p * 4) {
            let line = Path(CGRect(x: x, y: 0, width: 1, height: size.height))
            context.fill(line, with: .color(gridColor))
        }
        for y in stride(from: CGFloat(0), through: size.height, by: p * 4) {
            let line = Path(CGRect(x: 0, y: y, width: size.width, height: 1))
            context.fill(line, with: .color(gridColor))
        }

        switch zone {
        case .desk:
            drawDesk(in: context, size: size, p: p, palette: palette)
        case .library:
            drawLibrary(in: context, size: size, p: p, palette: palette)
        case .kitchen:
            drawKitchen(in: context, size: size, p: p, palette: palette)
        case .alertCorner:
            drawAlertCorner(in: context, size: size, p: p, palette: palette)
        }
    }

    private static func drawDesk(in context: GraphicsContext, size: CGSize, p: CGFloat, palette: ThemePalette) {
        let w = size.width
        // Desk surface at bottom
        let deskY = size.height - p * 8
        let desk = CGRect(x: p * 2, y: deskY, width: w - p * 4, height: p * 3)
        context.fill(Path(desk), with: .color(palette.furniture))
        // Desk legs
        context.fill(Path(CGRect(x: p * 3, y: deskY + p * 3, width: p * 1.5, height: p * 5)), with: .color(palette.furnitureDark))
        context.fill(Path(CGRect(x: w - p * 5, y: deskY + p * 3, width: p * 1.5, height: p * 5)), with: .color(palette.furnitureDark))
        // Plant (right area)
        let plantX = w - p * 16
        context.fill(Path(CGRect(x: plantX, y: deskY - p * 3, width: p * 2, height: p * 2)), with: .color(palette.plant))
        context.fill(Path(CGRect(x: plantX, y: deskY - p * 1, width: p * 2, height: p * 1)), with: .color(palette.pot))
        // Monitor (right side, gap after plant)
        let monX = w - p * 10
        context.fill(Path(CGRect(x: monX, y: deskY - p * 6, width: p * 8, height: p * 5)), with: .color(palette.screen))
        context.fill(Path(CGRect(x: monX + p * 0.5, y: deskY - p * 5.5, width: p * 7, height: p * 4)), with: .color(palette.screenGlow))
        // Monitor stand
        context.fill(Path(CGRect(x: monX + p * 3, y: deskY - p * 1, width: p * 2, height: p * 1)), with: .color(palette.furnitureDark))
    }

    private static func drawLibrary(in context: GraphicsContext, size: CGSize, p: CGFloat, palette: ThemePalette) {
        // Bookshelf against the back wall
        let shelfX = size.width - p * 14
        let shelfY = p * 4
        let shelfW = p * 12
        // Shelf frame
        context.fill(Path(CGRect(x: shelfX, y: shelfY, width: shelfW, height: size.height - p * 6)), with: .color(palette.furnitureDark))
        // Shelf rows with books
        let bookColors: [Color] = [palette.bookRed, palette.bookBlue, palette.bookGreen, palette.bookYellow]
        for row in 0..<3 {
            let rowY = shelfY + p * 2 + CGFloat(row) * p * 7
            // Shelf plank
            context.fill(Path(CGRect(x: shelfX + p, y: rowY + p * 5, width: shelfW - p * 2, height: p * 1)), with: .color(palette.furniture))
            // Books
            for book in 0..<5 {
                let bx = shelfX + p * 1.5 + CGFloat(book) * p * 2
                let bh = p * (3 + CGFloat(book % 2))
                let color = bookColors[(row * 5 + book) % bookColors.count]
                context.fill(Path(CGRect(x: bx, y: rowY + p * 5 - bh, width: p * 1.5, height: bh)), with: .color(color))
            }
        }
        // Floor rug
        context.fill(Path(roundedRect: CGRect(x: p * 2, y: size.height - p * 5, width: p * 10, height: p * 3), cornerRadius: p), with: .color(palette.rug))
    }

    private static func drawKitchen(in context: GraphicsContext, size: CGSize, p: CGFloat, palette: ThemePalette) {
        // Counter
        let counterY = size.height - p * 9
        context.fill(Path(CGRect(x: p * 2, y: counterY, width: size.width - p * 4, height: p * 2)), with: .color(palette.furniture))
        context.fill(Path(CGRect(x: p * 2, y: counterY + p * 2, width: size.width - p * 4, height: p * 7)), with: .color(palette.furnitureDark))
        // Cup + Steam (right area)
        let cupX = size.width - p * 18
        context.fill(Path(CGRect(x: cupX, y: counterY - p * 2, width: p * 2, height: p * 2)), with: .color(.white.opacity(0.8)))
        context.fill(Path(CGRect(x: cupX + p * 0.5, y: counterY - p * 3.5, width: p * 0.5, height: p * 1)), with: .color(.white.opacity(0.3)))
        context.fill(Path(CGRect(x: cupX + p * 1.5, y: counterY - p * 4, width: p * 0.5, height: p * 1)), with: .color(.white.opacity(0.2)))
        // Coffee machine (gap after cup)
        let cmX = size.width - p * 14
        context.fill(Path(CGRect(x: cmX, y: counterY - p * 5, width: p * 4, height: p * 5)), with: .color(palette.appliance))
        context.fill(Path(CGRect(x: cmX + p, y: counterY - p * 4, width: p * 2, height: p * 2)), with: .color(palette.screen))
        // Fridge (right side, original position)
        let fridgeX = size.width - p * 10
        context.fill(Path(CGRect(x: fridgeX, y: p * 3, width: p * 7, height: size.height - p * 11)), with: .color(palette.appliance))
        context.fill(Path(CGRect(x: fridgeX + p * 5.5, y: p * 6, width: p * 0.5, height: p * 3)), with: .color(palette.furnitureDark))
    }

    private static func drawAlertCorner(in context: GraphicsContext, size: CGSize, p: CGFloat, palette: ThemePalette) {
        // Warning stripes on floor
        for i in 0..<Int(size.width / (p * 3)) {
            let x = CGFloat(i) * p * 3
            if i % 2 == 0 {
                context.fill(Path(CGRect(x: x, y: size.height - p * 2, width: p * 3, height: p * 2)), with: .color(palette.alertStripe))
            }
        }
        // Alert board
        let boardX = size.width - p * 12
        context.fill(Path(CGRect(x: boardX, y: p * 3, width: p * 10, height: p * 8)), with: .color(palette.alertBoard))
        context.fill(Path(CGRect(x: boardX + p, y: p * 4, width: p * 8, height: p * 6)), with: .color(palette.alertBoardInner))
        // Warning triangle
        let triCenterX = boardX + p * 5
        let triY = p * 5
        var tri = Path()
        tri.move(to: CGPoint(x: triCenterX, y: triY))
        tri.addLine(to: CGPoint(x: triCenterX - p * 2, y: triY + p * 4))
        tri.addLine(to: CGPoint(x: triCenterX + p * 2, y: triY + p * 4))
        tri.closeSubpath()
        context.fill(tri, with: .color(palette.alertYellow))
        // Siren light
        context.fill(Path(CGRect(x: p * 4, y: p * 4, width: p * 3, height: p * 3)), with: .color(palette.alertRed.opacity(0.6)))
        context.fill(Path(CGRect(x: p * 4.5, y: p * 4.5, width: p * 2, height: p * 2)), with: .color(palette.alertRed))
    }

    struct ThemePalette {
        let floor, floorGrid: Color
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
    private static let cheek = Color(red: 0.96, green: 0.68, blue: 0.72)  // blush
    private static let belly = Color(red: 0.98, green: 0.95, blue: 0.88)
    private static let nose = Color(red: 0.92, green: 0.58, blue: 0.52)
    private static let eye = Color(red: 0.15, green: 0.12, blue: 0.12)
    private static let eyeShine = Color(red: 1.0, green: 1.0, blue: 1.0)
    private static let alert = Color(red: 0.98, green: 0.72, blue: 0.22)
    private static let error = Color(red: 0.88, green: 0.30, blue: 0.30)
    private static let success = Color(red: 0.35, green: 0.78, blue: 0.45)
    private static let paw = Color(red: 0.68, green: 0.50, blue: 0.38)

    // Symbol legend (10x10 grid):
    // F = fur, E = ear, B = belly, K = eye, W = eye shine, C = cheek/blush
    // N = nose, P = paw, A = alert/accessory, R = error, S = success, . = transparent

    static func renderToNSImage(frame: [String], hat: String, variant: String, size: NSSize) -> NSImage {
        let image = NSImage(size: size, flipped: false) { rect in
            guard let cgContext = NSGraphicsContext.current?.cgContext else { return false }
            let rows = frame.count
            let columns = frame.isEmpty ? 0 : frame[0].count
            guard rows > 0, columns > 0 else { return true }
            let pixelSize = min(rect.width / CGFloat(columns), rect.height / CGFloat(rows))
            let xOffset = (rect.width - pixelSize * CGFloat(columns)) / 2
            let yOffset = (rect.height - pixelSize * CGFloat(rows)) / 2

            for (rowIndex, row) in frame.enumerated() {
                for (columnIndex, symbol) in row.enumerated() {
                    guard let nsColor = nsColor(for: symbol, variant: variant) else { continue }
                    cgContext.setFillColor(nsColor)
                    let y = rect.height - yOffset - CGFloat(rowIndex + 1) * pixelSize
                    let pixelRect = CGRect(
                        x: xOffset + CGFloat(columnIndex) * pixelSize,
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
        case "A": return CGColor(red: 0.98, green: 0.72, blue: 0.22, alpha: 1)
        case "R": return CGColor(red: 0.88, green: 0.30, blue: 0.30, alpha: 1)
        case "S": return CGColor(red: 0.35, green: 0.78, blue: 0.45, alpha: 1)
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
        let pixelSize = min(size.width / CGFloat(columns), size.height / CGFloat(rows))
        let xOffset = (size.width - pixelSize * CGFloat(columns)) / 2
        let yOffset = (size.height - pixelSize * CGFloat(rows)) / 2

        for (rowIndex, row) in frame.enumerated() {
            for (columnIndex, symbol) in row.enumerated() {
                guard let color = color(for: symbol, variant: variant) else { continue }
                let rect = CGRect(
                    x: xOffset + CGFloat(columnIndex) * pixelSize,
                    y: yOffset + CGFloat(rowIndex) * pixelSize,
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

    // 8x8 chonky hamster bust shot — no legs, brown cap on head, white face+belly
    // B=white/cream F=light brown fur E=pink ear K=eye N=nose
    // A=accessory R=error S=success
    private static func framesFor(_ state: HamsterSpriteState) -> [[String]] {
        switch state {
        case .idle:
            return [[
                "..E..E..",
                ".FFFFFF.",
                "FBKBBKBF",
                "BBBNNBBB",
                "BBBBBBBB",
                "BBBBBBBB",
            ], [
                "..E..E..",
                ".FFFFFF.",
                "FBKBBKBF",
                "BBBNNBBB",
                "BBBBBBBB",
                "BBBBBBB.",
            ]]
        case .walk:
            return [[
                "..E..E..",
                ".FFFFFF.",
                "FBKBBKBF",
                "BBBNNBBB",
                "BBBBBBBB",
                "BBBBBBBB",
            ], [
                "..E..E..",
                ".FFFFFF.",
                "FBKBBKBF",
                "BBBNNBBB",
                "BBBBBBBB",
                ".BBBBBBB",
            ]]
        case .run:
            return [[
                "........",
                "..E..E..",
                ".FFFFFF.",
                "FBKBBKBF",
                "BBBNNBBB",
                "BBBBBBBB",
            ], [
                "..E..E..",
                ".FFFFFF.",
                "FBKBBKBF",
                "BBBNNBBB",
                "BBBBBBBB",
                "BBBBBBBB",
            ]]
        case .type:
            return [[
                "..E..E..",
                ".FFFFFF.",
                "FBKBBKBF",
                "BBBNNBBB",
                "BBBBBBBB",
                ".AAAAAA.",
            ], [
                "..E..E..",
                ".FFFFFF.",
                "FBKBBKBF",
                "BBBNNBBB",
                "BBBBBBBB",
                "A.AAAA.A",
            ]]
        case .read:
            return [[
                "..E..E..",
                ".FFFFFF.",
                "FBKBBKBF",
                "BBBNNBBB",
                "BBAABBBB",
                ".AAAAAA.",
            ]]
        case .think:
            return [[
                "..E..E.A",
                ".FFFFFF.",
                "FBKBBKBF",
                "BBBNNBBB",
                "BBBBBBBB",
                "BBBBBBBB",
            ], [
                "..E..EA.",
                ".FFFFFF.",
                "FBKBBKBF",
                "BBBNNBBB",
                "BBBBBBBB",
                "BBBBBBBB",
            ]]
        case .sleep:
            return [[
                "..E..E..",
                ".FFFFFF.",
                "FBFBBFBF",
                "BBBNNBBB",
                "BBBBBBBB",
                "BBBBBBBB",
            ], [
                "..E..E.A",
                ".FFFFFF.",
                "FBFBBFBF",
                "BBBNNBBB",
                "BBBBBBBB",
                "BBBBBBBB",
            ]]
        case .celebrate:
            return [[
                "S.E..E.S",
                ".FFFFFF.",
                "FBKBBKBF",
                "BBBNNBBB",
                "BBBBBBBB",
                "BBBBBBBB",
            ], [
                ".SE..ES.",
                ".FFFFFF.",
                "FBKBBKBF",
                "BBBNNBBB",
                "BBBBBBBB",
                "BBBBBBBB",
            ]]
        case .alert:
            return [[
                "...AA...",
                "..E..E..",
                ".FFFFFF.",
                "FBKBBKBF",
                "BBBNNBBB",
                "BBBBBBBB",
            ], [
                "..AAAA..",
                "..E..E..",
                ".FFFFFF.",
                "FBKBBKBF",
                "BBBNNBBB",
                "BBBBBBBB",
            ]]
        case .error:
            return [[
                "...RR...",
                "..E..E..",
                ".FFFFFF.",
                "FBKBBKBF",
                "BBBNNBBB",
                "BBBBBBBB",
            ], [
                "..RRRR..",
                "..E..E..",
                ".FFFFFF.",
                "FBKBBKBF",
                "BBBNNBBB",
                "BBBBBBBB",
            ]]
        }
    }
}
