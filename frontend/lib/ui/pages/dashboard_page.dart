import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:flutter/services.dart';
import 'package:dio/dio.dart';
import 'package:go_router/go_router.dart';
import 'dart:html' as html;
import '../../providers/auth_provider.dart';
import '../../providers/volume_provider.dart';
import '../../providers/file_provider.dart';
import '../../services/api_service.dart';

class DashboardPage extends StatefulWidget {
  const DashboardPage({super.key});

  @override
  State<DashboardPage> createState() => _DashboardPageState();
}

class _DashboardPageState extends State<DashboardPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<VolumeProvider>().fetchVolumes().then((_) {
        // 同步通知 FileProvider 默认选中的是哪一个
        final volProv = context.read<VolumeProvider>();
        if (volProv.selectedVolume != null) {
          context.read<FileProvider>().switchVolume(volProv.selectedVolume!.id);
        }
      });
    });
  }

  void _showAddVolumeDialog() {
    final nameCtrl = TextEditingController();
    final remarkCtrl = TextEditingController();
    showDialog(
      context: context,
      builder: (ctx) {
        return AlertDialog(
          backgroundColor: Colors.white,
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
          title: const Text('设立逻辑储存卷', style: TextStyle(fontWeight: FontWeight.w700)),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: nameCtrl,
                decoration: InputDecoration(
                  labelText: '卷名称 (例如：工作文档区)',
                  prefixIcon: const Icon(Icons.folder_shared_outlined),
                  border: OutlineInputBorder(borderRadius: BorderRadius.circular(12)),
                ),
              ),
              const SizedBox(height: 16),
              TextField(
                controller: remarkCtrl,
                decoration: InputDecoration(
                  labelText: '卷备注 (选填)',
                  prefixIcon: const Icon(Icons.edit_note),
                  border: OutlineInputBorder(borderRadius: BorderRadius.circular(12)),
                ),
              ),
            ],
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(ctx),
              child: const Text('取消', style: TextStyle(color: Colors.grey)),
            ),
            ElevatedButton(
              onPressed: () async {
                if (nameCtrl.text.isEmpty) return;
                final success = await context.read<VolumeProvider>().createVolume(nameCtrl.text, remarkCtrl.text);
                if (success && mounted) {
                  Navigator.pop(ctx);
                  ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('存储卷修建成功！')));
                }
              },
              style: ElevatedButton.styleFrom(
                backgroundColor: const Color(0xFF4F46E5),
                foregroundColor: Colors.white,
                shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
              ),
              child: const Text('开始部署'),
            ),
          ],
        );
      },
    );
  }

  void _showCreateFolderDialog() {
    final nameCtrl = TextEditingController();
    showDialog(
      context: context,
      builder: (ctx) {
        return AlertDialog(
          backgroundColor: Colors.white,
          title: const Text('新建文件夹'),
          content: TextField(
            controller: nameCtrl,
            decoration: const InputDecoration(labelText: '文件夹名称'),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(ctx),
              child: const Text('取消', style: TextStyle(color: Colors.grey)),
            ),
            ElevatedButton(
              style: ElevatedButton.styleFrom(backgroundColor: const Color(0xFF4F46E5), foregroundColor: Colors.white),
              onPressed: () async {
                if (nameCtrl.text.isNotEmpty) {
                  final succ = await context.read<FileProvider>().createFolder(nameCtrl.text);
                  if (mounted && succ) {
                    Navigator.pop(ctx);
                  }
                }
              },
              child: const Text('创建'),
            )
          ],
        );
      }
    );
  }

  Future<void> _handleDeleteFile(String path, bool isDir) async {
    final confirm = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        backgroundColor: Colors.white,
        title: const Text('确认销毁'),
        content: Text('您即将永久摧毁这个${isDir ? "文件夹及内部所有数据" : "文件"}，无法撤销。是否继续？'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          ElevatedButton(
            style: ElevatedButton.styleFrom(backgroundColor: Colors.redAccent, foregroundColor: Colors.white),
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('残忍删除'),
          ),
        ],
      ),
    );
    if (confirm == true && mounted) {
      await context.read<FileProvider>().deleteFile(path);
    }
  }

  void _showRenameDialog(String oldPath, String oldName) {
    final nameCtrl = TextEditingController(text: oldName);
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        backgroundColor: Colors.white,
        title: const Text('重塑名称'),
        content: TextField(
          controller: nameCtrl,
          decoration: const InputDecoration(labelText: '新名称'),
          autofocus: true,
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('取消')),
          ElevatedButton(
            style: ElevatedButton.styleFrom(backgroundColor: const Color(0xFF4F46E5), foregroundColor: Colors.white),
            onPressed: () async {
              if (nameCtrl.text.isNotEmpty && nameCtrl.text != oldName) {
                final succ = await context.read<FileProvider>().renameFile(oldPath, nameCtrl.text);
                if (mounted && succ) Navigator.pop(ctx);
              } else {
                Navigator.pop(ctx);
              }
            },
            child: const Text('确认修改'),
          )
        ],
      )
    );
  }

  void _showFileShareDialog(String path, bool isDir) {
    final pwdCtrl = TextEditingController();
    final customKeyCtrl = TextEditingController();
    int days = 0; // 0 = permanent
    String accessMode = 'public'; // public, password, login

    showDialog(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (context, setStateSB) => AlertDialog(
          backgroundColor: Colors.white,
          title: Text('创建${isDir ? "目录" : "文件"}分享'),
          content: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                const Text('您可以设置访问权限和有效期', style: TextStyle(fontSize: 12, color: Colors.grey)),
                const SizedBox(height: 16),
                DropdownButtonFormField<String>(
                  value: accessMode,
                  decoration: const InputDecoration(labelText: '访问权限', border: OutlineInputBorder()),
                  items: const [
                    DropdownMenuItem(value: 'public', child: Text('公开访问')),
                    DropdownMenuItem(value: 'password', child: Text('密码保护')),
                    DropdownMenuItem(value: 'login', child: Text('仅登录可见')),
                  ],
                  onChanged: (val) => setStateSB(() => accessMode = val!),
                ),
                if (accessMode == 'password') ...[
                  const SizedBox(height: 16),
                  TextField(
                    controller: pwdCtrl,
                    decoration: const InputDecoration(
                      labelText: '访问提取码',
                      prefixIcon: Icon(Icons.lock_person_outlined),
                      border: OutlineInputBorder(),
                    ),
                  ),
                ],
                const SizedBox(height: 16),
                DropdownButtonFormField<int>(
                  value: days,
                  decoration: const InputDecoration(labelText: '有效期', border: OutlineInputBorder()),
                  items: const [
                    DropdownMenuItem(value: 0, child: Text('永久有效')),
                    DropdownMenuItem(value: 1, child: Text('1 天')),
                    DropdownMenuItem(value: 7, child: Text('7 天')),
                    DropdownMenuItem(value: 30, child: Text('30 天')),
                  ],
                  onChanged: (val) => setStateSB(() => days = val!),
                ),
                if (days == 0) ...[
                  const SizedBox(height: 16),
                  TextField(
                    controller: customKeyCtrl,
                    decoration: const InputDecoration(
                      labelText: '自定义访问短码 (选填)',
                      hintText: '例如: my-cool-file',
                      prefixIcon: Icon(Icons.link_rounded),
                      border: OutlineInputBorder(),
                    ),
                  ),
                ],
              ],
            ),
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('取消')),
            ElevatedButton(
              style: ElevatedButton.styleFrom(backgroundColor: const Color(0xFF4F46E5), foregroundColor: Colors.white),
              onPressed: () async {
                final result = await context.read<FileProvider>().generateShare(
                  path, 
                  password: accessMode == 'password' ? pwdCtrl.text : null, 
                  days: days,
                  accessURLKey: days == 0 ? customKeyCtrl.text : null,
                  accessMode: accessMode,
                );
                // ... rest same
                if (mounted && result != null) {
                  Navigator.pop(ctx);
                  final key = result['key'];
                  final baseUrl = ApiService().dio.options.baseUrl.replaceAll('/api', '');
                  final shareUrl = '$baseUrl/f/$key';
                  _showShareResultDialog(shareUrl);
                }
              },
              child: const Text('生成并复制链接'),
            )
          ],
        ),
      ),
    );
  }

  void _showShareResultDialog(String url) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        backgroundColor: Colors.white,
        title: const Text('分享链接已就绪'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.check_circle, color: Colors.green, size: 48),
            const SizedBox(height: 16),
            SelectableText(url, style: const TextStyle(color: Colors.blue, fontWeight: FontWeight.bold)),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () {
              Clipboard.setData(ClipboardData(text: url));
              ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('链接已复制')));
              Navigator.pop(ctx);
            },
            child: const Text('复制链接'),
          ),
          ElevatedButton(onPressed: () => Navigator.pop(ctx), child: const Text('完成')),
        ],
      ),
    );
  }

  void _showPreview(String path, String name) {
    final ext = name.split('.').last.toLowerCase();
    final url = context.read<FileProvider>().getPreviewUrl(path);
    
    // For simplicity, we use different widgets based on extension
    Widget previewWidget;
    if (['jpg', 'jpeg', 'png', 'gif', 'webp'].contains(ext)) {
       previewWidget = Image.network(url, headers: {'Authorization': 'Bearer ${context.read<AuthProvider>().token}'});
    } else if (['txt', 'md', 'js', 'json', 'yaml', 'html', 'css', 'go', 'dart', 'py'].contains(ext)) {
       previewWidget = FutureBuilder<Response>(
         future: ApiService().dio.get(url.replaceFirst(ApiService().dio.options.baseUrl, '')),
         builder: (context, snapshot) {
           if (snapshot.connectionState == ConnectionState.waiting) return const Center(child: CircularProgressIndicator());
           if (snapshot.hasError) return Text('加载失败: ${snapshot.error}');
           return Container(
             color: Colors.grey[50],
             width: double.infinity,
             padding: const EdgeInsets.all(16),
             child: SingleChildScrollView(
               child: SelectableText(snapshot.data?.data.toString() ?? '', style: const TextStyle(fontFamily: 'monospace')),
             ),
           );
         },
       );
    } else {
       // Fallback to browser preview
       context.read<FileProvider>().downloadFile(path, preview: true);
       return;
    }

    showDialog(
      context: context,
      builder: (ctx) => Dialog(
        backgroundColor: Colors.white,
        child: Container(
          width: 800,
          height: 600,
          padding: const EdgeInsets.all(16),
          child: Column(
            children: [
              Row(
                children: [
                  const Icon(Icons.visibility_outlined, color: Colors.indigo),
                  const SizedBox(width: 8),
                  Text(name, style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 18)),
                  const Spacer(),
                  IconButton(onPressed: () => Navigator.pop(ctx), icon: const Icon(Icons.close)),
                ],
              ),
              const Divider(),
              Expanded(child: previewWidget),
            ],
          ),
        ),
      ),
    );
  }

  void _showVolumeAccessDialog() {
    final volProv = context.read<VolumeProvider>();
    final vol = volProv.selectedVolume;
    if (vol == null) return;

    String currentMode = vol.accessMode;
    final pwdCtrl = TextEditingController(text: '');
    final keyCtrl = TextEditingController(text: vol.accessURLKey);

    showDialog(
      context: context,
      builder: (ctx) {
        return StatefulBuilder(
          builder: (context, setStateSB) {
            final baseUrl = ApiService().dio.options.baseUrl.replaceAll('/api', '');
            final previewURL = '$baseUrl/s/${keyCtrl.text}';

            return AlertDialog(
              backgroundColor: Colors.white,
              title: const Text('设置卷访问权限'),
              content: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                   const Text('访问模式', style: TextStyle(fontWeight: FontWeight.bold)),
                  RadioListTile<String>(
                    title: const Text('私有 (仅自己)'),
                    value: 'private',
                    groupValue: currentMode,
                    onChanged: (v) => setStateSB(() => currentMode = v!),
                  ),
                  RadioListTile<String>(
                    title: const Text('完全开放 (任何人)'),
                    value: 'public',
                    groupValue: currentMode,
                    onChanged: (v) => setStateSB(() => currentMode = v!),
                  ),
                  RadioListTile<String>(
                    title: const Text('登录开放 (需账号)'),
                    value: 'login',
                    groupValue: currentMode,
                    onChanged: (v) => setStateSB(() => currentMode = v!),
                  ),
                  RadioListTile<String>(
                    title: const Text('密码访问 (知道密码)'),
                    value: 'password',
                    groupValue: currentMode,
                    onChanged: (v) => setStateSB(() => currentMode = v!),
                  ),
                  if (currentMode == 'password')
                    Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                      child: TextField(
                        controller: pwdCtrl,
                        obscureText: true,
                        decoration: const InputDecoration(
                          labelText: '设置访问密码',
                          border: OutlineInputBorder(),
                          prefixIcon: Icon(Icons.lock_outline),
                        ),
                      ),
                    ),
                  const Divider(),
                  const SizedBox(height: 8),
                  const Text('自定义访问短码 (Key)', style: TextStyle(fontWeight: FontWeight.bold)),
                  const SizedBox(height: 8),
                  TextField(
                    controller: keyCtrl,
                    onChanged: (v) => setStateSB(() {}),
                    decoration: const InputDecoration(
                      hintText: '例如：my-share',
                      border: OutlineInputBorder(),
                      prefixIcon: Icon(Icons.link),
                    ),
                  ),
                  if (currentMode != 'private')
                    Padding(
                      padding: const EdgeInsets.only(top: 12),
                      child: Row(
                        children: [
                          Expanded(
                            child: SelectableText(
                              '预览链接: $previewURL',
                              style: const TextStyle(fontSize: 12, color: Colors.blue, fontWeight: FontWeight.w500),
                            ),
                          ),
                          const SizedBox(width: 8),
                          IconButton(
                            icon: const Icon(Icons.copy_rounded, size: 18, color: Colors.blue),
                            tooltip: '复制到剪贴板',
                            onPressed: () {
                              Clipboard.setData(ClipboardData(text: previewURL));
                              ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('链接已复制到剪贴板！')));
                            },
                          ),
                          IconButton(
                            icon: const Icon(Icons.open_in_new_rounded, size: 18, color: Colors.blue),
                            tooltip: '立即跳转访问',
                            onPressed: () {
                               // 浏览器打开
                               html.window.open(previewURL, '_blank');
                            },
                          ),
                        ],
                      ),
                    ),
                ],
              ),
              actions: [
                TextButton(
                  onPressed: () => Navigator.pop(ctx),
                  child: const Text('取消', style: TextStyle(color: Colors.grey)),
                ),
                ElevatedButton(
                  style: ElevatedButton.styleFrom(backgroundColor: const Color(0xFF4F46E5), foregroundColor: Colors.white),
                  onPressed: () async {
                    if (currentMode == 'password' && pwdCtrl.text.isEmpty && vol.accessMode != 'password') {
                      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('开启密码模式必须配置口令')));
                      return;
                    }
                    if (keyCtrl.text.isEmpty) {
                       ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('访问短码不能为空')));
                       return;
                    }

                    final successData = await volProv.updateVolumeAccess(
                      vol.id, 
                      currentMode, 
                      pwdCtrl.text,
                      accessURLKey: keyCtrl.text,
                    );
                    
                    if (mounted && successData != null) {
                      Navigator.pop(ctx);
                      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('卷级权限与短码已更新')));
                    }
                  },
                  child: const Text('保存配置'),
                )
              ],
            );
          },
        );
      },
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: const Color(0xFFF3F4F6),
      body: Row(
        children: [
          // ================= 左侧导航区 =================
          Container(
            width: 260,
            color: Colors.white,
            child: Column(
              children: [
                Container(
                  padding: const EdgeInsets.all(24),
                  child: Row(
                    children: [
                      Container(
                        padding: const EdgeInsets.all(8),
                        decoration: BoxDecoration(color: const Color(0xFFEEF2FF), borderRadius: BorderRadius.circular(12)),
                        child: const Icon(Icons.cloud_outlined, size: 28, color: Color(0xFF4F46E5)),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Consumer<AuthProvider>(
                          builder: (context, auth, _) => Text(
                            auth.username ?? 'SysUser',
                            style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700),
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
                const Divider(height: 1, color: Color(0xFFE5E7EB)),
                // 分享管理入口
                Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                  child: ListTile(
                    leading: const Icon(Icons.share_outlined, color: Colors.indigo),
                    title: const Text('分享管理中心', style: TextStyle(fontWeight: FontWeight.w600, color: Colors.indigo)),
                    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                    onTap: () => context.push('/shares'),
                    hoverColor: Colors.indigo[50],
                  ),
                ),
                const Divider(height: 1, color: Color(0xFFE5E7EB)),
                // 动态卷列表区
                Expanded(
                  child: Consumer<VolumeProvider>(
                    builder: (context, volProv, _) {
                      if (volProv.isLoading && volProv.volumes.isEmpty) {
                        return const Center(child: CircularProgressIndicator());
                      }
                      if (volProv.volumes.isEmpty) {
                        return const Center(
                          child: Column(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Icon(Icons.inbox_rounded, size: 48, color: Colors.black12),
                              SizedBox(height: 16),
                              Text('这片存储域尚未垦殖', style: TextStyle(color: Colors.grey)),
                            ],
                          ),
                        );
                      }
                      return ListView.builder(
                        itemCount: volProv.volumes.length,
                        itemBuilder: (context, index) {
                          final vol = volProv.volumes[index];
                          final isSelected = volProv.selectedVolume?.id == vol.id;
                          return Padding(
                            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
                            child: ListTile(
                              leading: Icon(
                                isSelected ? Icons.folder_special : Icons.folder_outlined,
                                color: isSelected ? const Color(0xFF4F46E5) : const Color(0xFF6B7280),
                              ),
                              title: Text(
                                vol.name,
                                style: TextStyle(
                                  fontWeight: isSelected ? FontWeight.w600 : FontWeight.w500,
                                  color: isSelected ? const Color(0xFF4F46E5) : const Color(0xFF374151),
                                ),
                              ),
                              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                              selected: isSelected,
                              selectedTileColor: const Color(0xFFEEF2FF),
                              onTap: () {
                                volProv.selectVolume(vol);
                                context.read<FileProvider>().switchVolume(vol.id);
                              },
                            ),
                          );
                        },
                      );
                    },
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.all(20.0),
                  child: Column(
                    children: [
                      SizedBox(
                        width: double.infinity,
                        height: 48,
                        child: OutlinedButton.icon(
                          onPressed: _showAddVolumeDialog,
                          icon: const Icon(Icons.add_box_outlined),
                          label: const Text('开拓存储卷', style: TextStyle(fontWeight: FontWeight.w600)),
                          style: OutlinedButton.styleFrom(
                            foregroundColor: const Color(0xFF4F46E5),
                            side: const BorderSide(color: Color(0xFF4F46E5), width: 1.5),
                            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                          ),
                        ),
                      ),
                      const SizedBox(height: 12),
                      SizedBox(
                        width: double.infinity,
                        height: 48,
                        child: TextButton.icon(
                          onPressed: () => context.read<AuthProvider>().logout(),
                          icon: const Icon(Icons.power_settings_new_rounded, size: 20),
                          label: const Text('抛弃矩阵链路'),
                          style: TextButton.styleFrom(
                            foregroundColor: const Color(0xFF6B7280),
                            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
          Container(width: 1, color: const Color(0xFFE5E7EB)),
          
          // ================= 右侧信息主导区 =================
          Expanded(
            child: Column(
              children: [
                // 顶栏：显示当前卷信息和全量操作按钮
                Container(
                  height: 84,
                  padding: const EdgeInsets.symmetric(horizontal: 32),
                  decoration: const BoxDecoration(
                    color: Colors.white,
                    border: Border(bottom: BorderSide(color: Color(0xFFE5E7EB))),
                  ),
                  child: Row(
                    children: [
                      Consumer<VolumeProvider>(builder: (context, volProv, _) {
                        return Row(
                          children: [
                            Column(
                              mainAxisAlignment: MainAxisAlignment.center,
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text(
                                  volProv.selectedVolume?.name ?? '请跨越网关选择存储挂载区',
                                  style: Theme.of(context).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w700),
                                ),
                                if (volProv.selectedVolume?.remark.isNotEmpty ?? false)
                                  Text(volProv.selectedVolume!.remark, style: TextStyle(color: Colors.grey[500], fontSize: 12)),
                              ],
                            ),
                            if (volProv.selectedVolume != null) ...[
                              const SizedBox(width: 12),
                              IconButton(
                                icon: Icon(
                                  volProv.selectedVolume!.accessMode == 'private' ? Icons.lock_outline : Icons.public,
                                  color: volProv.selectedVolume!.accessMode == 'private' ? Colors.grey : Colors.green,
                                  size: 20,
                                ),
                                tooltip: '访问权限: ${volProv.selectedVolume!.accessMode}\n点击配置权限',
                                onPressed: _showVolumeAccessDialog,
                              )
                            ]
                          ],
                        );
                      }),
                      const Spacer(),
                      ElevatedButton.icon(
                        onPressed: _showCreateFolderDialog,
                        icon: const Icon(Icons.create_new_folder_outlined, size: 20),
                        label: const Text('新建目录'),
                        style: ElevatedButton.styleFrom(
                          backgroundColor: Colors.white,
                          foregroundColor: const Color(0xFF374151),
                          elevation: 0,
                          side: const BorderSide(color: Color(0xFFD1D5DB)),
                        ),
                      ),
                      const SizedBox(width: 12),
                      ElevatedButton.icon(
                        onPressed: () {
                           context.read<FileProvider>().uploadFiles();
                        },
                        icon: const Icon(Icons.cloud_upload_outlined, size: 20),
                        label: const Text('上传实体'),
                        style: ElevatedButton.styleFrom(
                          backgroundColor: const Color(0xFF4F46E5),
                          foregroundColor: Colors.white,
                          elevation: 0,
                        ),
                      ),
                    ],
                  ),
                ),
                // 右侧视图层与导航面包屑
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 32, vertical: 16),
                  alignment: Alignment.centerLeft,
                  child: Row(
                    children: [
                      IconButton(
                        onPressed: () => context.read<FileProvider>().goBack(),
                        icon: const Icon(Icons.arrow_upward_rounded, size: 20),
                        tooltip: '返回上层',
                      ),
                      const SizedBox(width: 8),
                      Consumer<FileProvider>(builder: (context, fileProv, _) {
                        final pathDisplay = fileProv.currentPath.isEmpty ? ' / (根目录)' : ' / ${fileProv.currentPath}';
                        return Text(
                          pathDisplay,
                          style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 16, color: Color(0xFF4B5563)),
                        );
                      }),
                      const Spacer(),
                      IconButton(
                        onPressed: () => context.read<FileProvider>().fetchFiles(),
                        icon: const Icon(Icons.refresh, size: 20, color: Colors.grey),
                        tooltip: '刷新',
                      ),
                    ],
                  ),
                ),
                // 卡片化文件列表
                Expanded(
                  child: Padding(
                    padding: const EdgeInsets.fromLTRB(32, 0, 32, 32),
                    child: Card(
                      child: Consumer<FileProvider>(
                        builder: (context, fileProv, _) {
                          if (fileProv.isLoading) {
                            return const Center(child: CircularProgressIndicator());
                          }
                          if (fileProv.files.isEmpty) {
                            return const Center(
                              child: Text('当前层级荒芜一物，可以上传文档或创建夹层', style: TextStyle(color: Colors.grey)),
                            );
                          }
                          return ListView.separated(
                            itemCount: fileProv.files.length,
                            separatorBuilder: (context, index) => const Divider(height: 1),
                            itemBuilder: (context, index) {
                              final f = fileProv.files[index];
                              return ListTile(
                                leading: Icon(
                                  f.isDir ? Icons.folder_rounded : Icons.insert_drive_file_rounded,
                                  color: f.isDir ? const Color(0xFFF59E0B) : const Color(0xFF6B7280),
                                  size: 32,
                                ),
                                title: Text(f.name, style: const TextStyle(fontWeight: FontWeight.w500)),
                                subtitle: Text(f.isDir ? '文件夹' : '${f.readableSize} • 刚刚', style: const TextStyle(fontSize: 12)),
                                trailing: Row(
                                  mainAxisSize: MainAxisSize.min,
                                  children: [
                                    if (!f.isDir)
                                      IconButton(
                                        icon: const Icon(Icons.download_rounded, color: Colors.blueGrey, size: 20),
                                        onPressed: () {
                                          fileProv.downloadFile(f.path);
                                        },
                                      ),
                                    PopupMenuButton<String>(
                                      onSelected: (val) {
                                        if (val == 'delete') _handleDeleteFile(f.path, f.isDir);
                                        if (val == 'rename') _showRenameDialog(f.path, f.name);
                                        if (val == 'share') _showFileShareDialog(f.path, f.isDir);
                                      },
                                      itemBuilder: (ctx) => [
                                        const PopupMenuItem(value: 'rename', child: Text('重命名')),
                                        const PopupMenuItem(value: 'share', child: Text('创建分享提取锁')),
                                        const PopupMenuItem(value: 'delete', child: Text('永久抹除', style: TextStyle(color: Colors.red))),
                                      ],
                                    ),
                                  ],
                                ),
                                onTap: () {
                                  if (f.isDir) {
                                    fileProv.enterDirectory(f.name);
                                  } else {
                                    _showPreview(f.path, f.name);
                                  }
                                },
                              );
                            },
                          );
                        },
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
