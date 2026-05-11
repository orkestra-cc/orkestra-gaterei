import 'package:orkestra_mobile/config/environment.dart';
import 'package:orkestra_mobile/config/app_config.dart';
import 'package:orkestra_mobile/main.dart' as app;

/// Development environment entry point.
///
/// Run with: flutter run -t lib/main_development.dart
void main() {
  AppConfig.initialize(EnvironmentConfig.development);
  app.main();
}
