import 'package:flutter/material.dart';
import 'package:dio/dio.dart';
import 'package:file_picker/file_picker.dart';
import 'package:shared_preferences/shared_preferences.dart';
// ignore: avoid_web_libraries_in_flutter
import 'dart:html' as html;
import '../services/api_service.dart';
import '../models/file_item.dart';

class FileProvider extends ChangeNotifier {
  List<FileItem> _files = [];
  bool _isLoading = false;
  
  int? _currentVolumeId;
  String _currentPath = '';

  List<FileItem> get files => _files;
  bool get isLoading => _isLoading;
  String get currentPath => _currentPath;

  // 接收从卷管理点击触发的切卷信号
  void switchVolume(int volumeId) {
    if (_currentVolumeId == volumeId) return;
    _currentVolumeId = volumeId;
    _currentPath = ''; // 归为根目录
    fetchFiles();
  }

  // 步入内层目录
  void enterDirectory(String folderName) {
    if (_currentPath.isEmpty) {
      _currentPath = folderName;
    } else {
      _currentPath = '$_currentPath/$folderName';
    }
    fetchFiles();
  }

  // 返回更高层级父目录
  void goBack() {
    if (_currentPath.isEmpty) return;
    final segments = _currentPath.split('/');
    if (segments.length <= 1) {
      _currentPath = '';
    } else {
      segments.removeLast();
      _currentPath = segments.join('/');
    }
    fetchFiles();
  }

  Future<void> fetchFiles() async {
    if (_currentVolumeId == null) return;
    
    _isLoading = true;
    notifyListeners();

    try {
      final res = await ApiService().dio.get('/v1/files/list', queryParameters: {
        'volumeId': _currentVolumeId,
        'path': _currentPath,
      });
      if (res.statusCode == 200) {
        final List data = res.data;
        _files = data.map((e) => FileItem.fromJson(e)).toList();
        
        // 我们在端侧强制把文件夹排在前面，文件排在后侧
        _files.sort((a, b) {
          if (a.isDir && !b.isDir) return -1;
          if (!a.isDir && b.isDir) return 1;
          return a.name.toLowerCase().compareTo(b.name.toLowerCase());
        });
      }
    } catch (e) {
      debugPrint('获取节点子项崩溃: $e');
      _files = [];
    }
    
    _isLoading = false;
    notifyListeners();
  }

  Future<bool> createFolder(String name) async {
    if (_currentVolumeId == null) return false;
    try {
      final res = await ApiService().dio.post('/v1/files/folder', data: {
        'volumeId': _currentVolumeId,
        'path': _currentPath,
        'name': name,
      });
      if (res.statusCode == 200) {
        fetchFiles();
        return true;
      }
    } catch (e) {
      debugPrint('创建物理文件夹失败: $e');
    }
    return false;
  }

  Future<bool> deleteFile(String path) async {
    if (_currentVolumeId == null) return false;
    try {
      final res = await ApiService().dio.delete('/v1/files/delete', data: {
        'volumeId': _currentVolumeId,
        'path': path,
      });
      if (res.statusCode == 200) {
        fetchFiles();
        return true;
      }
    } catch (e) {
      debugPrint('剥离文件映射失败: $e');
    }
    return false;
  }

  // 跨端接收本地磁盘文件后包装发出请求
  Future<void> uploadFiles() async {
    if (_currentVolumeId == null) return;
    
    try {
      FilePickerResult? result = await FilePicker.platform.pickFiles(allowMultiple: true);
      if (result != null) {
        _isLoading = true;
        notifyListeners();

        for (var file in result.files) {
          MultipartFile multipartFile;
          // 支持 Web 与 Desktop 双模
          if (file.bytes != null) {
             multipartFile = MultipartFile.fromBytes(file.bytes!, filename: file.name);
          } else if (file.path != null) {
             multipartFile = await MultipartFile.fromFile(file.path!, filename: file.name);
          } else {
             continue;
          }

          FormData formData = FormData.fromMap({
            "volumeId": _currentVolumeId.toString(),
            "path": _currentPath,
            "file": multipartFile,
          });

          await ApiService().dio.post('/v1/files/upload', data: formData);
        }
        
        await fetchFiles();
      }
    } catch (e) {
      debugPrint('多重数据链路传输崩溃了: $e');
      _isLoading = false;
      notifyListeners();
    }
  }

  /// 触发文件下载（Web 模式：通过带 token 的 URL + a 标签下载）
  Future<void> downloadFile(String filePath, {bool preview = false}) async {
    if (_currentVolumeId == null) return;
    final prefs = await SharedPreferences.getInstance();
    final token = prefs.getString('token') ?? '';
    final baseUrl = ApiService().dio.options.baseUrl;
    final url =
        '${baseUrl}/v1/files/download?volumeId=$_currentVolumeId&path=${Uri.encodeComponent(filePath)}&token=$token${preview ? "&preview=true" : ""}';
    
    if (preview) {
        // 预览模式直接新窗口打开
        html.window.open(url, '_blank');
        return;
    }

    // Web 平台：用 AnchorElement 触发下载
    final anchor = html.AnchorElement(href: url)
      ..setAttribute('download', '')
      ..style.display = 'none';
    html.document.body?.append(anchor);
    anchor.click();
    anchor.remove();
  }

  Future<bool> renameFile(String oldPath, String newName) async {
    if (_currentVolumeId == null) return false;
    try {
      final res = await ApiService().dio.put('/v1/files/rename', data: {
        'volumeId': _currentVolumeId,
        'oldPath': oldPath,
        'newName': newName,
      });
      if (res.statusCode == 200) {
        fetchFiles();
        return true;
      }
    } catch (e) {
      debugPrint('重命名失败: $e');
    }
    return false;
  }

  Future<Map<String, dynamic>?> generateShare(String path, {String? password, int days = 0, String? accessURLKey, String accessMode = 'public'}) async {
    if (_currentVolumeId == null) return null;
    try {
      final res = await ApiService().dio.post('/v1/share/generate', data: {
        'volumeId': _currentVolumeId,
        'path': path,
        'password': password,
        'days': days,
        'accessURLKey': accessURLKey,
        'accessMode': accessMode,
      });
      if (res.statusCode == 200) {
        return res.data;
      }
    } catch (e) {
      debugPrint('生成分享链接失败: $e');
    }
    return null;
  }

  String getPreviewUrl(String filePath) {
      if (_currentVolumeId == null) return '';
      final baseUrl = ApiService().dio.options.baseUrl;
      return '$baseUrl/v1/files/download?volumeId=$_currentVolumeId&path=${Uri.encodeComponent(filePath)}';
  }
}
