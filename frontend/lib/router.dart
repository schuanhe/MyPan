import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'providers/auth_provider.dart';
import 'ui/pages/login_page.dart';
import 'ui/pages/dashboard_page.dart';
import 'ui/pages/share_management_page.dart';

final rootNavigatorKey = GlobalKey<NavigatorState>();

GoRouter createRouter(AuthProvider authProvider) {
  return GoRouter(
    navigatorKey: rootNavigatorKey,
    initialLocation: '/login',
    refreshListenable: authProvider,
    redirect: (context, state) {
      final isGoingToLogin = state.matchedLocation == '/login';
      final isLoggedIn = authProvider.isAuthenticated;
      
      // 当处于应用冷启动校验 Token 读取阶段时，静待
      if (authProvider.isLoading) return null;

      if (!isLoggedIn && !isGoingToLogin) {
        return '/login';
      }
      if (isLoggedIn && isGoingToLogin) {
        return '/dashboard';
      }
      return null;
    },
    routes: [
      GoRoute(
        path: '/login',
        builder: (context, state) => LoginPage(
          redirect: state.uri.queryParameters['redirect'],
        ),
      ),
      GoRoute(
        path: '/dashboard',
        builder: (context, state) => const DashboardPage(),
      ),
      GoRoute(
        path: '/shares',
        builder: (context, state) => const ShareManagementPage(),
      ),
    ],
  );
}
