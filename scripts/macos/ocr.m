#import <Foundation/Foundation.h>
#import <CoreImage/CoreImage.h>
#import <Vision/Vision.h>

// Function declarations
CGImageRef convertCIImageToCGImage(CIImage *inputImage);
void recognizeText(NSURL *fileName, NSArray<NSString *> *recognitionLanguages);
BOOL detectQRCode(NSURL *fileName);

int main(int argc, const char * argv[]) {
    @autoreleasepool {
        if (@available(macOS 10.15, *)) {
            if (argc < 2) {
                NSLog(@"Usage: %s <imagePath> [languages]", argv[0]);
                return 1;
            }
            
            NSString *imagePath = [NSString stringWithUTF8String:argv[1]];
            NSMutableArray<NSString *> *languages = [NSMutableArray arrayWithArray:@[@"zh-Hans", @"en-US"]];
            
            if (argc > 2) {
                [languages removeAllObjects];
                for (int i = 2; i < argc; i++) {
                    NSString *language = [NSString stringWithUTF8String:argv[i]];
                    [languages addObject:language];
                }
            }

            NSURL *imageURL = [NSURL fileURLWithPath:imagePath];
            if (![[NSFileManager defaultManager] fileExistsAtPath:imagePath]) {
                NSLog(@"Invalid image path");
                return 1;
            }

            if (!detectQRCode(imageURL)) {
                recognizeText(imageURL, languages);
            }
        } else {
            NSLog(@"Only support macOS 10.15 and above");
        }
    }
    return 0;
}

CGImageRef convertCIImageToCGImage(CIImage *inputImage) {
    CIContext *context = [CIContext contextWithOptions:nil];
    return [context createCGImage:inputImage fromRect:inputImage.extent];
}

void recognizeText(NSURL *fileName, NSArray<NSString *> *recognitionLanguages) {
    if (@available(macOS 10.15, *)) {
        CIImage *ciImage = [CIImage imageWithContentsOfURL:fileName];
        CGImageRef img = convertCIImageToCGImage(ciImage);
        if (!ciImage || !img) {
            return;
        }
        
        VNImageRequestHandler *requestHandler = [[VNImageRequestHandler alloc] initWithCGImage:img options:@{}];
        
        VNRecognizeTextRequest *request = [[VNRecognizeTextRequest alloc] initWithCompletionHandler:^(VNRequest *request, NSError *error) {
            if (error) {
                fprintf(stderr, "Text recognition error: %s\n", error.localizedDescription.UTF8String);
                return;
            }

            NSArray<VNRecognizedTextObservation *> *observations = (NSArray<VNRecognizedTextObservation *> *)request.results;
            NSMutableArray<NSString *> *recognizedStrings = [NSMutableArray array];
            
            for (VNRecognizedTextObservation *observation in observations) {
                VNRecognizedText *topCandidate = [[observation topCandidates:1] firstObject];
                if (topCandidate) {
                    [recognizedStrings addObject:topCandidate.string];
                }
            }
            
            NSString *result = [recognizedStrings componentsJoinedByString:@" "];
            printf("%s\n", result.UTF8String);
        }];
        
        request.recognitionLanguages = recognitionLanguages;
        
        NSError *error = nil;
        if (![requestHandler performRequests:@[request] error:&error]) {
            fprintf(stderr, "Unable to perform the requests: %s\n", error.localizedDescription.UTF8String);
        }
    }
}

BOOL detectQRCode(NSURL *fileName) {
    CIImage *ciImage = [CIImage imageWithContentsOfURL:fileName];
    if (!ciImage) {
        return NO;
    }

    CIDetector *detector = [CIDetector detectorOfType:CIDetectorTypeQRCode context:nil options:@{CIDetectorAccuracy: CIDetectorAccuracyHigh}];
    NSArray *features = [detector featuresInImage:ciImage];

    BOOL isQRCode = NO;
    NSMutableString *result = [NSMutableString string];
    for (CIFeature *feature in features) {
        if ([feature.type isEqualToString:CIFeatureTypeQRCode]) {
            CIQRCodeFeature *qrFeature = (CIQRCodeFeature *)feature;
            isQRCode = YES;
            [result appendString:qrFeature.messageString ?: @""];
            [result appendString:@"\n"];
        }
    }

    if (isQRCode) {
        printf("%s\n", result.UTF8String);
    }

    return isQRCode;
}
