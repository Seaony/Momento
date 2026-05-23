// 中文注释：本扩展声明 Momento 资源库包的 UTType，供 Info.plist、打开面板和拖放识别共用。
import UniformTypeIdentifiers

extension UTType {
    static let momentoLibrary = UTType(exportedAs: "com.seaony.momento.library", conformingTo: .package)
}
