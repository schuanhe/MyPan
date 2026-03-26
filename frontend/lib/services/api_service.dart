import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';

class ApiService {
  static final ApiService _instance = ApiService._internal();

  factory ApiService() => _instance;

  late Dio dio;

  ApiService._internal() {
    dio = Dio(BaseOptions(
      // 调试模式下使用本地地址，发布模式下使用相对路径（由 Nginx 转发）
      baseUrl: kReleaseMode ? '/api' : 'http://localhost:8080/api',
      connectTimeout: const Duration(seconds: 5),
      receiveTimeout: const Duration(seconds: 15),
      responseType: ResponseType.json,
    ));

    dio.interceptors.add(InterceptorsWrapper(
      onRequest: (options, handler) async {
        final prefs = await SharedPreferences.getInstance();
        final token = prefs.getString('token');
        if (token != null && token.isNotEmpty) {
          options.headers['Authorization'] = 'Bearer $token'; // JWT 追加
        }
        return handler.next(options);
      },
      onError: (DioException e, handler) async {
        // 自主捕获一些特定的错误处理，比如 401 越权
        if (e.response?.statusCode == 401) {
          final prefs = await SharedPreferences.getInstance();
          await prefs.remove('token'); // 清除脏数据
        }
        return handler.next(e);
      },
    ));
  }
}
