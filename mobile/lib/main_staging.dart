import 'package:orkestra_mobile/config/environment.dart';
import 'package:orkestra_mobile/config/app_config.dart';
import 'package:orkestra_mobile/main.dart' as app;

/// Staging environment entry point.
///
/// Run with: flutter run -t lib/main_staging.dart
void main() {
  AppConfig.initialize(EnvironmentConfig.staging);
  app.main();
}
