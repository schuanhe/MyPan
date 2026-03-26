import 'package:flutter/material.dart';
import '../services/api_service.dart';
import '../models/volume.dart';

class VolumeProvider extends ChangeNotifier {
  List<Volume> _volumes = [];
  bool _isLoading = false;
  Volume? _selectedVolume;

  List<Volume> get volumes => _volumes;
  bool get isLoading => _isLoading;
  Volume? get selectedVolume => _selectedVolume;

  Future<void> fetchVolumes() async {
    _isLoading = true;
    Future.microtask(() => notifyListeners());

    try {
      final res = await ApiService().dio.get('/v1/volumes');
      if (res.statusCode == 200) {
        final List data = res.data;
        _volumes = data.map((e) => Volume.fromJson(e)).toList();

        if (_volumes.isNotEmpty && _selectedVolume == null) {
          _selectedVolume = _volumes.first;
        } else if (_selectedVolume != null) {
          // 刷新后同步已选中项的最新数据
          _selectedVolume = _volumes.firstWhere(
            (v) => v.id == _selectedVolume!.id,
            orElse: () => _volumes.first,
          );
        }
      }
    } catch (e) {
      debugPrint('容错拉取卷失败: $e');
    }

    _isLoading = false;
    notifyListeners();
  }

  void selectVolume(Volume vol) {
    _selectedVolume = vol;
    notifyListeners();
  }

  Future<bool> createVolume(String name, String remark) async {
    try {
      final res = await ApiService().dio.post('/v1/volumes', data: {
        'name': name,
        'remark': remark,
      });
      if (res.statusCode == 200) {
        await fetchVolumes();
        return true;
      }
    } catch (e) {
      debugPrint('建立存储卷异常: $e');
    }
    return false;
  }

  /// 设置开放卷访问权限
  /// [mode]: private / public / login / password
  Future<Map<String, dynamic>?> updateVolumeAccess(
      int volId, String mode, String password, {String? accessURLKey}) async {
    try {
      final res = await ApiService().dio.put('/v1/volumes/$volId/access', data: {
        'accessMode': mode,
        if (password.isNotEmpty) 'accessPassword': password,
        if (accessURLKey != null) 'accessURLKey': accessURLKey,
      });
      if (res.statusCode == 200) {
        await fetchVolumes();
        return res.data as Map<String, dynamic>;
      }
    } catch (e) {
      debugPrint('更新卷访问权限异常: $e');
    }
    return null;
  }
}
