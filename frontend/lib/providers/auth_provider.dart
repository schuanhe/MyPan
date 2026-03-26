import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../services/api_service.dart';

class AuthProvider extends ChangeNotifier {
  bool _isAuthenticated = false;
  String? _username;
  String? _token;
  bool _isLoading = true;

  bool get isAuthenticated => _isAuthenticated;
  String? get username => _username;
  String? get token => _token;
  bool get isLoading => _isLoading;

  Future<void> initAuth() async {
    final prefs = await SharedPreferences.getInstance();
    final token = prefs.getString('token'); // 是否缓存了合法的 Token
    
    if (token != null && token.isNotEmpty) {
      _isAuthenticated = true;
      _token = token;
      _username = prefs.getString('username');
    }
    
    _isLoading = false;
    notifyListeners();
  }

  Future<bool> login(String username, String password, {int? durationSeconds}) async {
    try {
      final res = await ApiService().dio.post('/auth/login', data: {
        'username': username,
        'password': password,
        'durationSeconds': durationSeconds,
      });

      if (res.statusCode == 200) {
        final token = res.data['token'];
        final user = res.data['user'];

        final prefs = await SharedPreferences.getInstance();
        await prefs.setString('token', token);
        await prefs.setString('username', user['username']);
        await prefs.setString('role', user['role']);

        _isAuthenticated = true;
        _token = token;
        _username = user['username'];
        notifyListeners();
        return true;
      }
    } catch (e) {
      debugPrint('Login exception: $e');
    }
    return false;
  }

  // 为无阻力地进入网盘默认创建一个便捷注册方案
  Future<bool> register(String inputUsername, String password) async {
    try {
      final res = await ApiService().dio.post('/auth/register', data: {
        'username': inputUsername,
        'password': password,
      });
      if (res.statusCode == 201) return true;
    } catch (e) {
      debugPrint('Registration exception: $e');
    }
    return false;
  }

  Future<void> logout() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.clear(); // 直接抹除当前域数据
    _isAuthenticated = false;
    _username = null;
    notifyListeners();
  }
}
