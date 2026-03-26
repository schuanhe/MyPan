import 'package:flutter/material.dart';
import '../services/api_service.dart';

class ShareItem {
  final int id;
  final String type; // "volume", "file", "directory"
  final String name;
  final String path;
  final String accessMode;
  final String accessURLKey;
  final DateTime? expiresAt;
  final int volumeId;
  final String volumeName;

  ShareItem({
    required this.id,
    required this.type,
    required this.name,
    required this.path,
    required this.accessMode,
    required this.accessURLKey,
    this.expiresAt,
    required this.volumeId,
    required this.volumeName,
  });

  factory ShareItem.fromJson(Map<String, dynamic> json) {
    return ShareItem(
      id: json['id'] ?? 0,
      type: json['type'] ?? '',
      name: json['name'] ?? '',
      path: json['path'] ?? '',
      accessMode: json['accessMode'] ?? '',
      accessURLKey: json['accessURLKey'] ?? '',
      expiresAt: json['expiresAt'] != null ? DateTime.parse(json['expiresAt']) : null,
      volumeId: json['volumeId'] ?? 0,
      volumeName: json['volumeName'] ?? '',
    );
  }
}

class ShareProvider extends ChangeNotifier {
  List<ShareItem> _shares = [];
  bool _isLoading = false;

  List<ShareItem> get shares => _shares;
  bool get isLoading => _isLoading;

  Future<void> fetchShares() async {
    _isLoading = true;
    notifyListeners();

    try {
      final res = await ApiService().dio.get('/v1/shares');
      if (res.statusCode == 200) {
        final List data = res.data;
        _shares = data.map((e) => ShareItem.fromJson(e)).toList();
      }
    } catch (e) {
      debugPrint('获取分享列表失败: $e');
    }

    _isLoading = false;
    notifyListeners();
  }

  Future<bool> revokeShare(String type, int id) async {
    try {
      final res = await ApiService().dio.delete('/v1/shares/$type/$id');
      if (res.statusCode == 200) {
        await fetchShares();
        return true;
      }
    } catch (e) {
      debugPrint('撤销分享失败: $e');
    }
    return false;
  }

  Future<bool> updateShare(int id, {String? password, int? days, String? accessURLKey, String? accessMode}) async {
    try {
      final res = await ApiService().dio.put('/v1/shares/file/$id', data: {
        'password': password,
        'days': days,
        'accessURLKey': accessURLKey,
        'accessMode': accessMode,
      });
      if (res.statusCode == 200) {
        await fetchShares();
        return true;
      }
    } catch (e) {
      debugPrint('更新分享失败: $e');
    }
    return false;
  }
}
