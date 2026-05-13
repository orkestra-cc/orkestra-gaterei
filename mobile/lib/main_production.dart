import 'package:orkestra_mobile/config/environment.dart';
import 'package:orkestra_mobile/config/app_config.dart';
import 'package:orkestra_mobile/main.dart' as app;

/// Production environment entry point.
///
/// Run with: flutter run -t lib/main_production.dart
/// Build with: flutter build apk -t lib/main_production.dart
void main() {
  AppConfig.initialize(EnvironmentConfig.production);
  app.main();
}
