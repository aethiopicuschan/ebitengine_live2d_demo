#include "LAppDefine.hpp"
#include "LAppAllocator.hpp"
#include "LAppTextureManager.hpp"
#include "LAppPal.hpp"
#include "CubismUserModelExtend.hpp"
#include "MouseActionManager.hpp"

#include <CubismFramework.hpp>
#include <Model/CubismUserModel.hpp>
#include <Physics/CubismPhysics.hpp>
#include <Rendering/OpenGL/CubismRenderer_OpenGLES2.hpp>
#include <Utils/CubismString.hpp>
#include <GL/glew.h>
#include <GLFW/glfw3.h>

struct Cubism
{
    Csm::csmFloat32 _userTimeSeconds;                               ///< デルタ時間の積算値[秒]
    Csm::csmVector<Csm::CubismIdHandle> _eyeBlinkIds;               ///< モデルに設定されたまばたき機能用パラメータID
    Csm::csmMap<Csm::csmString, Csm::ACubismMotion *> _motions;     ///< 読み込まれているモーションのリスト
    Csm::csmMap<Csm::csmString, Csm::ACubismMotion *> _expressions; ///< 読み込まれている表情のリスト

    Csm::CubismPose *_pose;                       ///< ポーズ管理
    Csm::CubismBreath *_breath;                   ///< 呼吸
    Csm::CubismPhysics *_physics;                 ///< 物理演算
    Csm::CubismEyeBlink *_eyeBlink;               ///< 自動まばたき
    Csm::CubismTargetPoint *_dragManager;         ///< マウスドラッグ
    Csm::CubismModelMatrix *_modelMatrix;         ///< モデル行列
    Csm::CubismMotionManager *_motionManager;     ///< モーション管理
    Csm::CubismMotionManager *_expressionManager; ///< 表情管理
    Csm::CubismModelUserData *_modelUserData;     ///< ユーザデータ

    Csm::csmFloat32 _dragX;         ///< マウスドラッグのX位置
    Csm::csmFloat32 _dragY;         ///< マウスドラッグのY位置
    Csm::csmFloat32 _accelerationX; ///< X軸方向の加速度
    Csm::csmFloat32 _accelerationY; ///< Y軸方向の加速度
    Csm::csmFloat32 _accelerationZ; ///< Z軸方向の加速度

    Csm::CubismUserModel *_userModel;
    Csm::CubismFramework::Option _cubismOption; ///< CubismFrameworkに関するオプション
    LAppAllocator _cubismAllocator;             ///< メモリのアロケーター
    int tps;                                    ///< TickPerSecond
    GLuint fbo;                                 ///< フレームバッファオブジェクト
    GLuint cb;                                  ///< カラーバッファ
    GLuint color_tex;
};

void InitializeCubism(Cubism *cubism)
{
    // setup cubism
    cubism->_cubismOption.LogFunction = LAppPal::PrintMessage;
    cubism->_cubismOption.LoggingLevel = Csm::CubismFramework::Option::LogLevel_Verbose;
    Csm::CubismFramework::StartUp(&cubism->_cubismAllocator, &cubism->_cubismOption);

    // Initialize cubism
    Csm::CubismFramework::Initialize();
}

extern "C"
{
    Cubism *new_cubism(int tps)
    {
        Cubism *c = new Cubism();

        c->tps = tps;

        // Cubism SDK の初期化
        InitializeCubism(c);

        return c;
    }

    // モデルをロードする
    void load_model(Cubism *cubism, char *_modelDirectoryName, char *_currentModelDirectory)
    {
        // モデルデータの新規生成
        std::string modelDirectoryName(_modelDirectoryName);
        std::string currentModelDirectory(_currentModelDirectory);
        cubism->_userModel = new CubismUserModelExtend(modelDirectoryName, currentModelDirectory);

        // モデルデータの読み込み及び生成とセットアップを行う
        std::string json = ".model3.json";
        std::string fileName = modelDirectoryName + json;
        static_cast<CubismUserModelExtend *>(cubism->_userModel)->LoadAssets(fileName.c_str());
    }

    int get_drawable_count(Cubism *cubism)
    {
        return cubism->_userModel->GetModel()->GetDrawableCount();
    }

    int get_texture_index(Cubism *cubism, int drawableIndex)
    {
        return cubism->_userModel->GetModel()->GetDrawableTextureIndex(drawableIndex);
    }

    int get_drawable_vertex_count(Cubism *cubism, int drawableIndex)
    {
        return cubism->_userModel->GetModel()->GetDrawableVertexCount(drawableIndex);
    }

    const Live2D::Cubism::Core::csmVector2 *get_drawable_vertex_positions(Cubism *cubism, int drawableIndex)
    {
        return cubism->_userModel->GetModel()->GetDrawableVertexPositions(drawableIndex);
    }

    const Live2D::Cubism::Core::csmVector2 *get_drawable_vertex_uvs(Cubism *cubism, int drawableIndex)
    {
        return cubism->_userModel->GetModel()->GetDrawableVertexUvs(drawableIndex);
    }

    int get_index_count(Cubism *cubism, int drawableIndex)
    {
        return cubism->_userModel->GetModel()->GetDrawableVertexIndexCount(drawableIndex);
    }

    const uint16_t *get_drawable_vertex_indices(Cubism *cubism, int drawableIndex)
    {
        return cubism->_userModel->GetModel()->GetDrawableVertexIndices(drawableIndex);
    }

    const Live2D::Cubism::Framework::CubismId *get_drawable_id(Cubism *cubism, int drawableIndex)
    {
        return cubism->_userModel->GetModel()->GetDrawableId(drawableIndex);
    }

    float get_drawable_opacity(Cubism *cubism, int drawableIndex)
    {
        return cubism->_userModel->GetModel()->GetDrawableOpacity(drawableIndex);
    }

    const int *get_drawable_render_orders(Cubism *cubism)
    {
        return cubism->_userModel->GetModel()->GetDrawableRenderOrders();
    }

    int get_drawable_blend_mode(Cubism *cubism, int drawableIndex)
    {
        return cubism->_userModel->GetModel()->GetDrawableBlendMode(drawableIndex);
    }

    int get_mask_count(Cubism *cubism, int drawableIndex)
    {
        return cubism->_userModel->GetModel()->GetDrawableMaskCounts()[drawableIndex];
    }

    const int *get_masks(Cubism *cubism, int drawableIndex)
    {
        return cubism->_userModel->GetModel()->GetDrawableMasks()[drawableIndex];
    }

    bool get_drawable_is_inverted_mask(Cubism *cubism, int drawableIndex)
    {
        return cubism->_userModel->GetModel()->GetDrawableInvertedMask(drawableIndex);
    }

    int get_drawable_culling(Cubism *cubism, int drawableIndex)
    {
        return cubism->_userModel->GetModel()->GetDrawableCulling(drawableIndex);
    }

    bool get_drawable_dynamic_flag_is_visible(Cubism *cubism, int drawableIndex)
    {
        return cubism->_userModel->GetModel()->GetDrawableDynamicFlagIsVisible(drawableIndex);
    }

    bool get_drawable_dynamic_flag_vertex_positions_did_change(Cubism *cubism, int drawableIndex)
    {
        return cubism->_userModel->GetModel()->GetDrawableDynamicFlagVertexPositionsDidChange(drawableIndex);
    }

    const char *get_texture_file_name(Cubism *cubism, int textureIndex)
    {
        return static_cast<CubismUserModelExtend *>(cubism->_userModel)->_textureManager->GetTextureInfoByIndex(textureIndex)->fileName.c_str();
    }

    void update(Cubism *cubism)
    {
        LAppPal::UpdateTime2(cubism->tps);
        static_cast<CubismUserModelExtend *>(cubism->_userModel)->ModelOnUpdate2(cubism->_userModel->GetModel()->GetCanvasWidthPixel(), cubism->_userModel->GetModel()->GetCanvasHeightPixel());
    }

    float get_pixels_per_unit(Cubism *cubism)
    {
        return cubism->_userModel->GetModel()->GetPixelsPerUnit();
    }

    float get_canvas_width(Cubism *cubism)
    {
        return cubism->_userModel->GetModel()->GetCanvasWidth();
    }

    float get_canvas_height(Cubism *cubism)
    {
        return cubism->_userModel->GetModel()->GetCanvasHeight();
    }

    float get_canvas_width_pixel(Cubism *cubism)
    {
        return cubism->_userModel->GetModel()->GetCanvasWidthPixel();
    }

    float get_canvas_height_pixel(Cubism *cubism)
    {
        return cubism->_userModel->GetModel()->GetCanvasHeightPixel();
    }

    int get_parameter_count(Cubism *cubism)
    {
        return cubism->_userModel->GetModel()->GetParameterCount();
    }

    const char *get_parameter_id(Cubism *cubism, int parameterIndex)
    {
        const Csm::CubismId *id = cubism->_userModel->GetModel()->GetParameterId(parameterIndex);
        return id->GetString().GetRawString();
    }

    void add_parameter_value(Cubism *cubism, int parameterIndex, float value)
    {
        cubism->_userModel->GetModel()->AddParameterValue(parameterIndex, value);
        cubism->_userModel->GetModel()->Update();
    }
}
