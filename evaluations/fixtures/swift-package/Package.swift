// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "TextKit",
    platforms: [
        .macOS(.v13),
        .iOS(.v16),
    ],
    products: [
        .library(name: "TextKit", targets: ["TextKit"]),
    ],
    targets: [
        .target(name: "TextKit"),
        .testTarget(name: "TextKitTests", dependencies: ["TextKit"]),
    ]
)
