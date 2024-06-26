import Foundation
import CoreImage
import Vision
import ArgumentParser

let joiner = " "

// Define the arguments accepted by the program
struct OCRArguments: ParsableCommand {
  @Argument(help: "The path to the image file.")
  var imagePath: String

  @Option(help: "Optional language codes for text recognition. Separate by spaces (e.g., 'zh-Hans en-US').")
  var languages: [String] = ["zh-Hans", "en-US"]
}

func convertCIImageToCGImage(inputImage: CIImage) -> CGImage? {
  let context = CIContext(options: nil)
  return context.createCGImage(inputImage, from: inputImage.extent)
}

@available(OSX 10.15, *)
func recognizeTextHandler(request: VNRequest, error: Error?) {
  guard let observations = request.results as? [VNRecognizedTextObservation] else {
      return
  }
  let recognizedStrings = observations.compactMap { observation in
      // Return the string of the top VNRecognizedText instance.
      observation.topCandidates(1).first?.string
  }

  // Process the recognized strings.
  let result = recognizedStrings.joined(separator: joiner)

  print(result)
}

@available(OSX 10.15, *)
func detectText(fileName: URL, recognitionLanguages: [String]) {
  guard
          let ciImage = CIImage(contentsOf: fileName),
          let img = convertCIImageToCGImage(inputImage: ciImage)
  else {
      return
  }

  let requestHandler = VNImageRequestHandler(cgImage: img)

  // Create a new request to recognize text.
  let request = VNRecognizeTextRequest(completionHandler: recognizeTextHandler)
  request.recognitionLanguages = recognitionLanguages

  do {
      // Perform the text-recognition request.
      try requestHandler.perform([request])
  } catch {
      print("Unable to perform the requests: \(error).")
  }
}

func detectImage(fileName: URL) -> Bool {
  let detector = CIDetector(ofType: CIDetectorTypeQRCode, context: nil, options: nil)

  guard let ciImage = CIImage(contentsOf: fileName), let features = detector?.features(in: ciImage) else {
      return false
  }

  var isQRCode = false
  var result = ""
  for feature in features as! [CIQRCodeFeature] {
      if feature.type == "QRCode" {
          isQRCode = true
      }
      result += feature.messageString ?? ""
      result += "\n"
  }

  if isQRCode {
      print(result)
  }

  return isQRCode
}

func main() {
  // 只支持 macOS 10.15 以上
  guard #available(OSX 10.15, *) else {
    return print("Only support the system above macOS 10.15")
  }

    do {
        let arguments = try OCRArguments.parse()
        
        let imagePath = URL(fileURLWithPath: arguments.imagePath)
        
        if imagePath.path.isEmpty {
            print("读取图片路径出错")
            return
        }
        
        // 检测是否有二维码，直接将二维码转换成解码后的内容
        guard detectImage(fileName: imagePath) else {
            return detectText(fileName: imagePath, recognitionLanguages: arguments.languages)
        }
    } catch {
        print("解析命令行错误")
    }
}
