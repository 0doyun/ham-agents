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

struct PixelOfficeView: View {
    let occupants: [PixelOfficeOccupant]
    let animationSpeedMultiplier: Double
    let reduceMotion: Bool
    let hamsterSkin: String
    let hat: String
    let deskTheme: String

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Pixel Office")
                .font(.caption.weight(.semibold))

            HStack(alignment: .top, spacing: 8) {
                zoneCard(.desk, title: "Desk")
                zoneCard(.library, title: "Library")
            }
            HStack(alignment: .top, spacing: 8) {
                zoneCard(.kitchen, title: "Kitchen")
                zoneCard(.alertCorner, title: "Alert")
            }
        }
    }

    @ViewBuilder
    private func zoneCard(_ zone: PixelOfficeZone, title: String) -> some View {
        let zoneOccupants = occupants.filter { $0.zone == zone }

        VStack(alignment: .leading, spacing: 6) {
            Text(title)
                .font(.caption2.weight(.bold))
                .foregroundStyle(.secondary)

            if zoneOccupants.isEmpty {
                Spacer(minLength: 0)
                Text("—")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            } else {
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 34), spacing: 4)], spacing: 4) {
                    ForEach(zoneOccupants) { occupant in
                        VStack(spacing: 2) {
                            PixelHamsterSpriteView(
                                state: occupant.sprite,
                                variant: occupant.agent.avatarVariant == "default" ? hamsterSkin : occupant.agent.avatarVariant,
                                hat: hat,
                                animationSpeedMultiplier: animationSpeedMultiplier,
                                reduceMotion: reduceMotion
                            )
                            .frame(width: 24, height: 24)

                            Text(occupant.agent.displayName)
                                .font(.system(size: 8, weight: .medium))
                                .lineLimit(1)
                        }
                    }
                }
            }
        }
        .padding(8)
        .frame(maxWidth: .infinity, minHeight: 86, alignment: .topLeading)
        .background(zoneBackground(for: zone))
        .clipShape(RoundedRectangle(cornerRadius: 10))
    }

    private func zoneBackground(for zone: PixelOfficeZone) -> Color {
        let palette: (Color, Color, Color, Color)
        switch deskTheme {
        case "night-shift":
            palette = (Color.indigo.opacity(0.16), Color.purple.opacity(0.16), Color.teal.opacity(0.16), Color.red.opacity(0.18))
        case "sunny":
            palette = (Color.yellow.opacity(0.16), Color.orange.opacity(0.14), Color.green.opacity(0.12), Color.pink.opacity(0.14))
        default:
            palette = (Color.blue.opacity(0.08), Color.purple.opacity(0.08), Color.green.opacity(0.08), Color.orange.opacity(0.12))
        }
        switch zone {
        case .desk:
            return palette.0
        case .library:
            return palette.1
        case .kitchen:
            return palette.2
        case .alertCorner:
            return palette.3
        }
    }
}

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

enum PixelHamsterLibrary {
    private static let furDefault = Color(red: 0.67, green: 0.52, blue: 0.40)
    private static let furNight = Color(red: 0.55, green: 0.43, blue: 0.34)
    private static let furGolden = Color(red: 0.87, green: 0.72, blue: 0.37)
    private static let furMint = Color(red: 0.56, green: 0.74, blue: 0.64)
    private static let ear = Color(red: 0.96, green: 0.76, blue: 0.80)
    private static let belly = Color(red: 0.97, green: 0.95, blue: 0.88)
    private static let eye = Color.black
    private static let alert = Color(red: 0.98, green: 0.64, blue: 0.16)
    private static let error = Color(red: 0.88, green: 0.28, blue: 0.28)
    private static let success = Color(red: 0.26, green: 0.76, blue: 0.41)

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
                    // Flip Y: NSImage drawing rep is bottom-up
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
            case "golden": return CGColor(red: 0.87, green: 0.72, blue: 0.37, alpha: 1)
            case "mint":   return CGColor(red: 0.56, green: 0.74, blue: 0.64, alpha: 1)
            case "night":  return CGColor(red: 0.55, green: 0.43, blue: 0.34, alpha: 1)
            default:       return CGColor(red: 0.67, green: 0.52, blue: 0.40, alpha: 1)
            }
        case "N": return CGColor(red: 0.55, green: 0.43, blue: 0.34, alpha: 1)
        case "E": return CGColor(red: 0.96, green: 0.76, blue: 0.80, alpha: 1)
        case "B": return CGColor(red: 0.97, green: 0.95, blue: 0.88, alpha: 1)
        case "K": return CGColor(red: 0, green: 0, blue: 0, alpha: 1)
        case "A": return CGColor(red: 0.98, green: 0.64, blue: 0.16, alpha: 1)
        case "R": return CGColor(red: 0.88, green: 0.28, blue: 0.28, alpha: 1)
        case "S": return CGColor(red: 0.26, green: 0.76, blue: 0.41, alpha: 1)
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
            case "golden":
                return furGolden
            case "mint":
                return furMint
            case "night":
                return furNight
            default:
                return furDefault
            }
        case "N":
            return furNight
        case "E":
            return ear
        case "B":
            return belly
        case "K":
            return eye
        case "A":
            return alert
        case "R":
            return error
        case "S":
            return success
        default:
            return nil
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
        let width = pixelSize * 4
        let rect = CGRect(x: xOffset + pixelSize * 2, y: yOffset + pixelSize * 0.5, width: width, height: pixelSize * 1.2)
        context.fill(Path(roundedRect: rect, cornerRadius: pixelSize * 0.4), with: .color(color))
    }

    private static func frames(for state: HamsterSpriteState) -> [[String]] {
        switch state {
        case .idle:
            return framesFor(.idle)
        case .walk:
            return framesFor(.walk)
        case .run:
            return framesFor(.run)
        case .type:
            return framesFor(.type)
        case .read:
            return framesFor(.read)
        case .think:
            return framesFor(.think)
        case .sleep:
            return framesFor(.sleep)
        case .celebrate:
            return framesFor(.celebrate)
        case .alert:
            return framesFor(.alert)
        case .error:
            return framesFor(.error)
        }
    }

    private static func framesFor(_ state: HamsterSpriteState) -> [[String]] {
        switch state {
        case .idle:
            return [[
                "........",
                "..EE....",
                ".EFFE...",
                ".FFFB...",
                ".FFKBB..",
                "..FBB...",
                ".BBBB...",
                "........",
            ]]
        case .walk:
            return [[
                "........",
                "..EE....",
                ".EFFE...",
                ".FFFB...",
                ".FFKBB..",
                "..FBB...",
                ".B..B...",
                "........",
            ], [
                "........",
                "..EE....",
                ".EFFE...",
                ".FFFB...",
                ".FFKBB..",
                "..FBB...",
                "..BB.B..",
                "........",
            ]]
        case .run:
            return [[
                "........",
                "..EE....",
                ".EFFE...",
                ".FFFB...",
                ".FFKBB..",
                ".FFBB...",
                ".B..B...",
                "..B.B...",
            ], [
                "........",
                "..EE....",
                ".EFFE...",
                ".FFFB...",
                ".FFKBB..",
                ".FFBB...",
                "..BB....",
                ".B..B...",
            ]]
        case .type:
            return [[
                "........",
                "..EE....",
                ".EFFEAA.",
                ".FFFB.A.",
                ".FFKBBA.",
                "..FBB...",
                ".BBBB...",
                "........",
            ], [
                "........",
                "..EE....",
                ".EFFEAA.",
                ".FFFB..A",
                ".FFKBBA.",
                "..FBB...",
                ".BBBB...",
                "........",
            ]]
        case .read:
            return [[
                "...AA...",
                "..EEA...",
                ".EFFEA..",
                ".FFFBA..",
                ".FFKBB..",
                "..FBB...",
                ".BBBB...",
                "........",
            ]]
        case .think:
            return [[
                "...AA...",
                "..EE.A..",
                ".EFFE...",
                ".FFFB...",
                ".FFKBB..",
                "..FBB...",
                ".BBBB...",
                "........",
            ], [
                "....AA..",
                "..EE....",
                ".EFFE.A.",
                ".FFFB...",
                ".FFKBB..",
                "..FBB...",
                ".BBBB...",
                "........",
            ]]
        case .sleep:
            return [[
                "........",
                "..EE....",
                ".EFFE...",
                ".FFFF...",
                ".FBBBB..",
                "..BBB...",
                "...AA...",
                "........",
            ]]
        case .celebrate:
            return [[
                ".S....S.",
                "..EE....",
                ".EFFE...",
                ".FFFB...",
                ".FFKBB..",
                "..FBB...",
                ".B..B...",
                "S....S..",
            ]]
        case .alert:
            return [[
                "...A....",
                "..EA....",
                ".EFFE...",
                ".FFFB...",
                ".FFKBB..",
                "..FBB...",
                ".BBBB...",
                "........",
            ]]
        case .error:
            return [[
                "...R....",
                "..ER....",
                ".EFFE...",
                ".FFFR...",
                ".FFKBB..",
                "..FBB...",
                ".BBBB...",
                "........",
            ]]
        }
    }
}
